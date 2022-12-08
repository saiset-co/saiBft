package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/bft/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

var (
	errNotEnoughVotes = errors.New("tx msg with certain number of votes was not found")
)

const (
	blockchainCol      = "Blockchain"
	MessagesPoolCol    = "MessagesPool"
	ConsensusPoolCol   = "ConsensusPool"
	BlockCandidatesCol = "BlockCandidates"
	maxRoundNumber     = 7
	btcKeyFile         = "btc_keys.json"
)

// main process of blockchain
func (s *InternalService) Processing() {
	saiBtcAddress, ok := s.GlobalService.Configuration["saiBTC_address"].(string)
	if !ok {
		s.GlobalService.Logger.Fatal("processing - wrong type of saiBTC address value from config")
	}
	saiP2Paddress, ok := s.GlobalService.Configuration["saiP2P_address"].(string)
	if !ok {
		s.GlobalService.Logger.Fatal("processing - wrong type of saiP2P address value from config")
	}

	s.GlobalService.Logger.Sugar().Debugf("starting processing") //DEBUG

	// for tests
	//btcKeys1, _ := s.getBTCkeys("btc_keys3.json", saiBtcAddress)
	// btcKeys2, _ := s.getBTCkeys("btc_keys2.json", saiBtcAddress)
	// btcKeys3, _ := s.getBTCkeys("btc_keys1.json", saiBtcAddress)
	//s.TrustedValidators = append(s.TrustedValidators, s.BTCkeys.Address)
	//
	//s.GlobalService.Logger.Sugar().Debugf("btc keys : %+v\n", s.BTCkeys) //DEBUG
	//
	storageToken, ok := s.GlobalService.Configuration["storage_token"].(string)
	if !ok {
		s.GlobalService.Logger.Fatal("handlers - processing - wrong type of storage token value from config")
	}

	// get trusted validators from config
	trustedValidatorsInterface, ok := s.GlobalService.Configuration["trusted_validators"].([]interface{})
	if !ok {
		s.GlobalService.Logger.Fatal("handlers - processing - wrong type of trusted_validators value from config")
	}

	for _, validator := range trustedValidatorsInterface {
		s.TrustedValidators = append(s.TrustedValidators, validator.(string))
	}

	s.GlobalService.Logger.Sugar().Debugf("got trusted validators : %v", s.TrustedValidators) //DEBUG

	s.GlobalService.Logger.Debug("Start node mode", zap.Bool("skip initializating", s.SkipInitializating)) //DEBUG

	// initial block consensus waiting
	if !s.SkipInitializating {
		initialBCTimeout, ok := s.GlobalService.Configuration["initial_block_consensus_timeout_sec"].(int)
		if !ok {
			s.GlobalService.Logger.Fatal("processing - wrong type of inital block consensus timeout  value from config")
		}

		s.GlobalService.Logger.Debug("initial block consensus initializing", zap.Int("timeout in seconds", initialBCTimeout)) //DEBUG
		initialTimer := time.NewTimer(time.Duration(initialBCTimeout * int(time.Second)))
	initialBlockConsensus:
		for {
			select {
			case <-s.InitialSignalCh:
				initialTimer.Stop()
				s.IsInitialized = true
				s.GlobalService.Logger.Debug("node was initialized by incoming block consensus msg")
				break initialBlockConsensus
			case <-initialTimer.C:
				initialTimer.Stop()
				s.IsInitialized = true
				s.GlobalService.Logger.Debug("initializing timeout expired")
				break initialBlockConsensus
			}
		}
	}

	//TEST transaction &consensus messages
	s.saveTestTx(saiBtcAddress, storageToken, saiP2Paddress)

	for {
	startLoop:
		round := 0
		s.GlobalService.Logger.Debug("start loop,round = 0") // DEBUG
		time.Sleep(1 * time.Second)                          //DEBUG

		// get last block from blockchain collection or create initial block
		block, err := s.getLastBlockFromBlockChain(storageToken, saiBtcAddress)
		if err != nil {
			continue
		}
	checkRound:
		select {
		case <-s.GoToStartLoopCh:
			s.GlobalService.Logger.Debug("signal from chain was got, move to startLoop")
			goto startLoop
		default:
		}
		s.GlobalService.Logger.Sugar().Debugf("ROUND = %d", round) //DEBUG
		if round == 0 {
			// get messages with votes = 0
			transactions, err := s.getZeroVotedTransactions(storageToken)
			if err != nil {
				s.GlobalService.Logger.Error("process - round == 0 - get zero-voted tx messages", zap.Error(err))
			}
			consensusMsg := &models.ConsensusMessage{
				Type:          models.ConsensusMsgType,
				SenderAddress: s.BTCkeys.Address,
				BlockNumber:   block.Block.Number,
				Round:         round,
			}
			// validate/execute each tx msg, update hash and votes
			if len(transactions) != 0 {
				for _, tx := range transactions {
					err = s.validateExecuteTransactionMsg(tx, saiBtcAddress, storageToken)
					if err != nil {
						continue
					}
					consensusMsg.Messages = append(consensusMsg.Messages, tx.ExecutedHash)
				}
			}

			consensusMsg.Round = round + 1

			consensusMsg.Hash, err = consensusMsg.GetHash()
			if err != nil {
				s.GlobalService.Logger.Error("process - round==0 - hash consensus message", zap.Error(err))
				goto startLoop
			}

			btcResp, err := utils.SignMessage(consensusMsg, saiBtcAddress, s.BTCkeys.Private)
			if err != nil {
				s.GlobalService.Logger.Error("process - round==0 - sign consensus message", zap.Error(err))
				goto startLoop
			}
			consensusMsg.Signature = btcResp.Signature

			err, _ = s.Storage.Put("ConsensusPool", consensusMsg, storageToken)
			if err != nil {
				s.GlobalService.Logger.Error("process - round == 0 - put consensus to ConsensusPool collection", zap.Error(err))
				goto startLoop
			}

			err = s.broadcastMsg(consensusMsg, saiP2Paddress)
			if err != nil {
				s.GlobalService.Logger.Error("process - round==0 - broadcast consensus message", zap.Error(err))
				goto startLoop
			}

			time.Sleep(time.Duration(s.GlobalService.Configuration["sleep"].(int)) * time.Second)
			round++
			goto checkRound

		} else {
			// get consensus messages for the round
			msgs, err := s.getConsensusMsgForTheRound(round, block.Block.Number, storageToken)
			if err != nil {
				goto startLoop
			}
			for _, msg := range msgs {
				// check if consensus message sender is from trusted validators list
				err = checkConsensusMsgSender(s.TrustedValidators, msg)
				if err != nil {
					s.GlobalService.Logger.Error("process - round != 0 - check consensus message sender", zap.Error(err))
					continue
				}

				s.GlobalService.Logger.Sugar().Debugf("Consensus message transactions: %v", msg.Messages) //DEBUG

				// update votes for each tx message from consensusMsg
				for _, txMsgHash := range msg.Messages {
					err, result := s.Storage.Get("MessagesPool", bson.M{"executed_hash": txMsgHash}, bson.M{}, storageToken)
					if err != nil {
						s.GlobalService.Logger.Error("process - get msg from consensus msg from storage", zap.Error(err))
						continue
					}

					if len(result) == 2 {
						continue
					}
					err = s.updateTxMsgVotes(txMsgHash, storageToken, round)
					if err != nil {
						continue
					}
				}
			}

			// get messages with votes>=(roundNumber*10)%
			txMsgs, err := s.getTxMsgsWithCertainNumberOfVotes(storageToken, round)
			if err != nil {
				goto startLoop
			}

			if round < maxRoundNumber-1 {
				newConsensusMsg := &models.ConsensusMessage{
					Type:          models.ConsensusMsgType,
					SenderAddress: Service.BTCkeys.Address,
					BlockNumber:   block.Block.Number,
				}

				for _, txMsg := range txMsgs {
					newConsensusMsg.Messages = append(newConsensusMsg.Messages, txMsg.ExecutedHash)
				}
				newConsensusMsg.Round = round + 1

				newConsensusMsgHash, err := newConsensusMsg.GetHash()
				if err != nil {
					s.GlobalService.Logger.Error("process - round==0 - hash consensus message", zap.Error(err))
					goto startLoop
				}

				newConsensusMsg.Hash = newConsensusMsgHash

				btcResp, err := utils.SignMessage(newConsensusMsg, saiBtcAddress, s.BTCkeys.Private)
				if err != nil {
					s.GlobalService.Logger.Error("process - round==0 - sign consensus message", zap.Error(err))
					goto startLoop
				}

				newConsensusMsg.Signature = btcResp.Signature

				err, _ = s.Storage.Put("ConsensusPool", newConsensusMsg, storageToken)
				if err != nil {
					s.GlobalService.Logger.Error("process - round == 0 - put consensus to ConsensusPool collection", zap.Error(err))
					goto startLoop
				}

				err = s.broadcastMsg(newConsensusMsg, saiP2Paddress)
				if err != nil {
					goto startLoop
				}
			}

			round++

			time.Sleep(time.Duration(s.GlobalService.Configuration["sleep"].(int)) * time.Second)

			if round < maxRoundNumber {
				goto checkRound
			} else {
				s.GlobalService.Logger.Sugar().Debugf("ROUND = %d", round) //DEBUG

				newBlock, err := s.formAndSaveNewBlock(block, saiBtcAddress, storageToken, txMsgs)
				if err != nil {
					goto startLoop
				}
				err = s.broadcastMsg(newBlock, saiP2Paddress)
				if err != nil {
					goto startLoop
				}

				goto startLoop
			}
		}
	}
}

// get last block from blockchain collection
func (s *InternalService) getLastBlockFromBlockChain(storageToken string, saiBtcAddress string) (*models.BlockConsensusMessage, error) {
	opts := options.Find().SetSort(bson.M{"block.number": -1}).SetLimit(1)
	err, result := s.Storage.Get(blockchainCol, bson.M{}, opts, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("handlers - process - processing - get last block", zap.Error(err))
		return nil, err
	}

	// empty get response returns '{}' in storage get method -> new block should be created
	if len(result) == 2 {
		block, err := s.createInitialBlock(saiBtcAddress)
		if err != nil {
			s.GlobalService.Logger.Error("process - create initial block", zap.Error(err))
			return nil, err
		}
		return block, nil
	} else {
		blocks := make([]*models.BlockConsensusMessage, 0)
		data, err := utils.ExtractResult(result)
		if err != nil {
			Service.GlobalService.Logger.Error("process - get last block from blockchain - extract data from response", zap.Error(err))
			return nil, err
		}
		s.GlobalService.Logger.Sugar().Debugf("get last block data : %s", string(data))
		err = json.Unmarshal(data, &blocks)
		if err != nil {
			s.GlobalService.Logger.Error("handlers - process - unmarshal result of last block from blockchain collection", zap.Error(err))
			return nil, err
		}
		block := blocks[0]
		block.Block.Number++
		s.GlobalService.Logger.Sugar().Debugf("Got last block from blockchain collection : %+v\n", block) //DEBUG

		return block, nil
	}
}

// create initial block
func (s *InternalService) createInitialBlock(address string) (block *models.BlockConsensusMessage, err error) {
	s.GlobalService.Logger.Sugar().Debugf("block not found, creating initial block") //DEBUG

	block = &models.BlockConsensusMessage{
		Type: models.BlockConsensusMsgType,
		Block: &models.Block{
			Number:            1,
			SenderAddress:     s.BTCkeys.Address,
			PreviousBlockHash: "",
			Messages:          make(map[string]*models.Tx),
		},
	}

	blockHash, err := block.Block.GetHash()
	if err != nil {
		return nil, err
	}
	block.BlockHash = blockHash
	block.Block.BlockHash = blockHash

	btcResp, err := utils.SignMessage(block, address, s.BTCkeys.Private)
	if err != nil {
		return nil, err
	}
	block.Block.SenderSignature = btcResp.Signature

	s.GlobalService.Logger.Sugar().Debugf("First block created : %+v\n", block) //DEBUG

	return block, nil

}

// get messages with votes = 0
func (s *InternalService) getZeroVotedTransactions(storageToken string) ([]*models.TransactionMessage, error) {
	err, result := s.Storage.Get("MessagesPool", bson.M{"votes.0": 0}, bson.M{}, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("process - round = 0 - get messages with 0 votes", zap.Error(err))
		return nil, err
	}

	if len(result) == 2 {
		err = errors.New("no 0 voted messages found")
		s.GlobalService.Logger.Error("process - round = 0 - get messages with 0 votes", zap.Error(err))
		return nil, err
	}

	data, err := utils.ExtractResult(result)
	if err != nil {
		Service.GlobalService.Logger.Error("process - getZeroVotedTransactions - extract data from response", zap.String("data", string(result)), zap.Error(err))
		return nil, err
	}

	transactions := make([]*models.TransactionMessage, 0)

	err = json.Unmarshal(data, &transactions)
	if err != nil {
		s.GlobalService.Logger.Error("handlers - process - round = 0 - unmarshal result of messages with votes = 0", zap.Error(err))
		return nil, err
	}

	filteredTx := make([]*models.TransactionMessage, 0)

	for _, tx := range transactions {
		if tx.BlockNumber == 0 && tx.BlockHash == "" {
			filteredTx = append(filteredTx, tx)
		}
	}

	s.GlobalService.Logger.Sugar().Debugf("Got transactions with votes = 0 : %+v", filteredTx) //DEBUG

	return filteredTx, nil
}

// validate/execute each message, update message and hash and vote for valid messages
func (s *InternalService) validateExecuteTransactionMsg(msg *models.TransactionMessage, saiBTCaddress, storageToken string) error {
	s.GlobalService.Logger.Sugar().Debugf("Handling transaction : %+v", msg) //DEBUG

	err := utils.ValidateSignature(msg, saiBTCaddress, msg.Tx.SenderAddress, msg.Tx.SenderSignature)
	if err != nil {
		s.GlobalService.Logger.Error("process - ValidateExecuteTransactionMsg - validate tx msg signature", zap.Error(err))
		return err
	}

	// dummy vm result values after executing at vm
	msg.VmResult = true
	msg.VmResponse = "vmResponse"

	err = msg.GetExecutedHash()
	if err != nil {
		s.GlobalService.Logger.Error("process - ValidateExecuteTransactionMsg - get executed hash", zap.Error(err))
		return err
	}

	msg.Votes[0]++
	filter := bson.M{"message_hash": msg.MessageHash}
	update := bson.M{"votes": msg.Votes, "vm_processed": true, "vm_result": msg.VmResult, "vm_response": msg.VmResponse, "executed_hash": msg.ExecutedHash}
	err, _ = s.Storage.Update("MessagesPool", filter, update, storageToken)
	if err != nil {
		Service.GlobalService.Logger.Error("process - ValidateExecuteTransactionMsg - update transactions in storage", zap.Error(err))
		return err
	}
	return nil

}

// check if consensus message sender is not from validators list
func checkConsensusMsgSender(validators []string, msg *models.ConsensusMessage) error {
	for _, validator := range validators {
		if msg.SenderAddress == validator {
			return nil
		}
	}
	return fmt.Errorf("Consensus message sender is not from validators list, validators : %s, sender : %s", validators, msg.SenderAddress)
}

// get consensus messages for the round
func (s *InternalService) getConsensusMsgForTheRound(round, blockNumber int, storageToken string) ([]*models.ConsensusMessage, error) {
	err, result := s.Storage.Get("ConsensusPool", bson.M{"round": round, "block_number": blockNumber}, bson.M{}, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("process - round != 0 - get messages for specified round", zap.Error(err))
		return nil, err
	}

	if len(result) == 2 {
		err = fmt.Errorf("no consensusMsg found for round : %d", round)
		s.GlobalService.Logger.Error("process - get consensusMsg for round", zap.Int("round", round), zap.Error(err))
		return nil, err
	}

	data, err := utils.ExtractResult(result)
	if err != nil {
		Service.GlobalService.Logger.Error("process - getConsensusMsgForTheRound - extract data from response", zap.Error(err))
		return nil, err
	}

	msgs := make([]*models.ConsensusMessage, 0)

	err = json.Unmarshal(data, &msgs)
	if err != nil {
		s.GlobalService.Logger.Error("process - round != 0 - unmarshal result of consensus messages for specified round", zap.Error(err))
		return nil, err
	}

	return msgs, nil
}

// broadcast messages to connected nodes
func (s *InternalService) broadcastMsg(msg interface{}, saiP2Paddress string) error {
	param := url.Values{}
	data, err := json.Marshal(msg)

	if err != nil {
		s.GlobalService.Logger.Error("process - round != 0 - broadcastMsg - marshal msg", zap.Error(err))
		return err
	}

	param.Add("message", string(data))
	postRequest, err := http.NewRequest("POST", saiP2Paddress+"/Send_message", bytes.NewBufferString(param.Encode()))
	if err != nil {
		s.GlobalService.Logger.Error("process - round != 0 - broadcastMsg - create post request", zap.Error(err))
		return err
	}

	postRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(postRequest)
	if err != nil {
		s.GlobalService.Logger.Error("process - round != 0 - broadcastMsg - send post request ", zap.String("host", saiP2Paddress), zap.Error(err))
		return err
	}

	defer resp.Body.Close()

	// if resp.StatusCode != 200 {
	// 	s.GlobalService.Logger.Sugar().Errorf("process - round != 0 - broadcastMsg - send post request to host : %s wrong status code : %d", saiP2Paddress, resp.StatusCode)
	// 	return fmt.Errorf("Wrong status code : %d", resp.StatusCode)
	// }

	s.GlobalService.Logger.Sugar().Debugf("broadcasting - success, message : %+v", msg) // DEBUG                                                       //DEBUG
	return nil
}

// form and save new block
func (s *InternalService) formAndSaveNewBlock(previousBlock *models.BlockConsensusMessage, saiBTCaddress, storageToken string, txMsgs []*models.TransactionMessage) (*models.BlockConsensusMessage, error) {
	newBlock := &models.BlockConsensusMessage{
		Type: models.BlockConsensusMsgType,
		Block: &models.Block{
			Number:            previousBlock.Block.Number,
			PreviousBlockHash: previousBlock.BlockHash,
			SenderAddress:     s.BTCkeys.Address,
			Messages:          make(map[string]*models.Tx),
		},
	}

	for _, tx := range txMsgs {
		newBlock.Block.Messages[tx.MessageHash] = tx.Tx
	}

	blockHash, err := newBlock.Block.GetHash()
	if err != nil {
		s.GlobalService.Logger.Error("process - round == 7 - form and save new block - count hash of new block", zap.Error(err))
		return nil, err
	}
	newBlock.BlockHash = blockHash
	newBlock.Block.BlockHash = blockHash

	btcResp, err := utils.SignMessage(newBlock, saiBTCaddress, s.BTCkeys.Private)

	if err != nil {
		s.GlobalService.Logger.Error("process - round == 7 - form and save new block - sign message", zap.Error(err))
		return nil, err

	}

	newBlock.Votes = +1
	newBlock.Block.SenderSignature = btcResp.Signature
	newBlock.Signatures = append(newBlock.Signatures, btcResp.Signature)

	// check if we have already such block candidate
	blockCandidate, err := s.getBlockCandidate(newBlock.BlockHash, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("process - round == 7 - form and save new block - sign message", zap.Error(err))
		return nil, err
	}

	// if there is no such blockCandidate, save block to BlockCandidate collection
	if blockCandidate == nil {
		err, _ := s.Storage.Put(BlockCandidatesCol, newBlock, storageToken)
		if err != nil {
			s.GlobalService.Logger.Error("process - round == 7 - form and save new block - put to BlockCandidate collection", zap.Error(err))
			return nil, err
		}

	} else { // else, add vote and signature and save to blockchain
		s.GlobalService.Logger.Debug("found block candidate with formed block hash", zap.String("hash", newBlock.BlockHash))
		newBlock.Votes = newBlock.Votes + blockCandidate.Votes
		newBlock.Signatures = append(newBlock.Signatures, blockCandidate.Signatures...)

		err, _ = s.Storage.Remove(BlockCandidatesCol, bson.M{"hash": newBlock.BlockHash}, storageToken)
		if err != nil {
			s.GlobalService.Logger.Error("process - round == 7 - form and save new block - remove block from block candidates", zap.Error(err))
			return nil, err
		}
		err, _ = s.Storage.Put(blockchainCol, newBlock, storageToken)
		if err != nil {
			s.GlobalService.Logger.Error("process - round == 7 - form and save new block - put block to blockchain collection", zap.Error(err))
			return nil, err
		}
	}

	for _, tx := range txMsgs {
		err, _ := s.Storage.Update("MessagesPool", bson.M{"message_hash": tx.MessageHash}, bson.M{"block_hash": newBlock.BlockHash, "block_number": newBlock.Block.Number}, storageToken)
		if err != nil {
			s.GlobalService.Logger.Error("process - round == 7 - form and save new block - update tx blockhash", zap.Error(err))
			return nil, err
		}
	}

	err = s.updateTxMsgZeroVotes(storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("process - round == 7  - form and save new block - clear messages", zap.Error(err))
		return nil, err
	}

	s.GlobalService.Logger.Sugar().Debugf(" formed new block to save: %+v\n", newBlock) //DEBUG

	return newBlock, nil
}

// update votes to zero for transaction message
func (s *InternalService) updateTxMsgZeroVotes(storageToken string) error {
	criteria := bson.M{"votes.0": bson.M{"$gte": 1}, "block_hash": ""}
	update := bson.M{"votes": bson.A{0, 0, 0, 0, 0, 0, 0}}

	_, result := s.Storage.Get("MessagesPool", criteria, bson.M{}, storageToken)
	s.GlobalService.Logger.Sugar().Debugf("BEFORE UPDATING ON CLEAR RESULT : %s", string(result))

	err, _ := s.Storage.Update("MessagesPool", criteria, update, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("handlers - process - round != 0 - get messages for specified round", zap.Error(err))
		return err
	}

	criteria2 := bson.M{"block_hash": ""}
	_, result2 := s.Storage.Get("MessagesPool", criteria2, bson.M{}, storageToken)
	s.GlobalService.Logger.Sugar().Debugf("AFTER UPDATING ON CLEAR RESULT : %s", string(result2))

	return nil
}

// update votes for transaction message
func (s *InternalService) updateTxMsgVotes(hash, storageToken string, round int) error {
	// DEBUG votes updating
	// criteria := bson.M{"message_hash": hash}
	// _, result := s.Storage.Get("MessagesPool", criteria, bson.M{}, storageToken)
	// s.GlobalService.Logger.Sugar().Debugf("BEFORE UPDATING ON ROUND : %d, RESULT : %s", round, string(result))
	/////

	criteria := bson.M{"executed_hash": hash}
	update := bson.M{"$inc": bson.M{"votes." + strconv.Itoa(round): 1}}
	err, _ := s.Storage.Upsert("MessagesPool", criteria, update, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("handlers - process - round != 0 - get messages for specified round", zap.Error(err))
		return err
	}

	// DEBUG votes updating
	// criteria = bson.M{"message_hash": hash}
	// _, result = s.Storage.Get("MessagesPool", criteria, bson.M{}, storageToken)
	// s.GlobalService.Logger.Sugar().Debugf("AFTER UPDATING ON ROUND : %d, RESULT : %s", round, string(result))
	/////
	return nil
}

// get messages with certain number of votes
func (s *InternalService) getTxMsgsWithCertainNumberOfVotes(storageToken string, round int) ([]*models.TransactionMessage, error) {
	requiredVotes := math.Ceil(float64(len(s.TrustedValidators)) * float64(round) * 10 / 100)
	filterGte := bson.M{"votes." + strconv.Itoa(round): bson.M{"$gte": requiredVotes}}
	filteredTx := make([]*models.TransactionMessage, 0)
	txMsgs := make([]*models.TransactionMessage, 0)

	err, result := s.Storage.Get("MessagesPool", filterGte, bson.M{}, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("handlers - process - round != 0 - get tx messages with specified votes count", zap.Float64("votes count", requiredVotes), zap.Error(err))
		return nil, err
	}

	if len(result) == 2 {
		s.GlobalService.Logger.Error("process - get tx msgs with certain number of votes - emtpy result", zap.String("required votes", strconv.Itoa(int(requiredVotes))))
		return filteredTx, nil
	}

	data, err := utils.ExtractResult(result)
	if err != nil {
		Service.GlobalService.Logger.Error("process - getZeroVotedTransactions - extract data from response", zap.String("result", string(result)), zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(data, &txMsgs)
	if err != nil {
		s.GlobalService.Logger.Error("process - round != 0 - unmarshal result of consensus messages for specified round", zap.Error(err))
		return nil, err
	}

	for _, tx := range txMsgs {
		if tx.BlockHash == "" {
			filteredTx = append(filteredTx, tx)
		}
	}
	return filteredTx, nil
}

func (s *InternalService) GetBTCkeys(fileStr, saiBTCaddress string) (*models.BtcKeys, error) {
	file, err := os.OpenFile(fileStr, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		s.GlobalService.Logger.Error("processing - open key btc file", zap.Error(err))
		return nil, err
	}

	data, err := ioutil.ReadFile(fileStr)
	if err != nil {
		s.GlobalService.Logger.Error("processing - read key btc file", zap.Error(err))
		return nil, err
	}
	btcKeys := models.BtcKeys{}

	err = json.Unmarshal(data, &btcKeys)
	if err != nil {
		s.GlobalService.Logger.Error("get btc keys - error unmarshal from file", zap.Error(err))
		btcKeys, body, err := utils.GetBtcKeys(saiBTCaddress)
		if err != nil {
			s.GlobalService.Logger.Error("processing - get btc keys", zap.Error(err))
			return nil, err
		}
		_, err = file.Write(body)
		if err != nil {
			s.GlobalService.Logger.Error("processing - write btc keys to file", zap.Error(err))
			return nil, err
		}
		return btcKeys, nil
	} else {
		err = btcKeys.Validate()
		if err != nil {
			btcKeys, body, err := utils.GetBtcKeys(saiBTCaddress)
			if err != nil {
				s.GlobalService.Logger.Fatal("processing - get btc keys", zap.Error(err))
				return nil, err
			}
			_, err = file.Write(body)
			if err != nil {
				s.GlobalService.Logger.Fatal("processing - write btc keys to file", zap.Error(err))
				return nil, err
			}
			return btcKeys, nil
		}
		return &btcKeys, nil
	}
}

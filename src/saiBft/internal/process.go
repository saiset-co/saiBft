package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/bft/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	blockchainCollection = "Blockchain"
	maxRoundNumber       = 7
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
	time.Sleep(time.Second)                                      //DEBUG

	// for tests
	btcKeys1, _ := s.getBTCkeys("btc_keys3.json", saiBtcAddress)
	// btcKeys2, _ := s.getBTCkeys("btc_keys2.json", saiBtcAddress)
	// btcKeys3, _ := s.getBTCkeys("btc_keys1.json", saiBtcAddress)
	s.TrustedValidators = append(s.TrustedValidators, btcKeys1.Address)

	btckeys, _ := s.getBTCkeys("btc_keys.json", saiBtcAddress)
	s.BTCkeys = btckeys
	s.GlobalService.Logger.Sugar().Debugf("btc keys : %+v\n", s.BTCkeys) //DEBUG
	time.Sleep(time.Second)                                              //DEBUG

	storageToken, ok := s.GlobalService.Configuration["storage_token"].(string)
	if !ok {
		s.GlobalService.Logger.Fatal("handlers - processing - wrong type of storage token value from config")
	}

	// get trusted validators from config

	// trustedValidorsInterface, ok := s.GlobalService.Configuration["trusted_validators"].([]interface{})
	// if !ok {
	// 	s.GlobalService.Logger.Fatal("handlers - processing - wrong type of storage token value from config")
	// }

	// for _, validator := range trustedValidorsInterface {
	// 	s.TrustedValidators = append(s.TrustedValidators, validator.(string))

	// }

	s.GlobalService.Logger.Sugar().Debugf("got trusted validators : %v", s.TrustedValidators) //DEBUG
	time.Sleep(time.Second)                                                                   //DEBUG

	//TEST transaction
	s.saveTestTx(saiBtcAddress, storageToken)
	s.GlobalService.Logger.Sugar().Debugf("test tx message saved") //DEBUG
	time.Sleep(time.Second)                                        //DEBUG

	// TEST consensusMsg
	s.saveTestConsensusMsg(saiBtcAddress, storageToken, s.TrustedValidators[0])
	s.GlobalService.Logger.Sugar().Debugf("test consensus message saved") //DEBUG
	time.Sleep(time.Second)                                               //DEBUG

	for {
		round := 0
	startLoop:
		time.Sleep(3 * time.Second) // for test purposes
		s.GlobalService.Logger.Debug("start loop,round = 0")
		time.Sleep(time.Second) //DEBUG

		// for testing purposes
		time.Sleep(2 * time.Second)

		block := &models.BlockConsensusMessage{}

		// get last block from blockchain collection or create initial block
		result, err := s.getLastBlockFromBlockChain(storageToken)
		if err != nil {
			continue
		}

		// empty get response returns '{}' in storage get method -> new block should be created
		if len(result) == 2 {
			s.GlobalService.Logger.Sugar().Debugf("block not found, creating initial block")
			block, err = s.createInitialBlock(saiBtcAddress)
			if err != nil {
				s.GlobalService.Logger.Error("process - create initial block", zap.Error(err))
				continue
			}

			s.GlobalService.Logger.Sugar().Debugf("First block created : %+v\n", block) //DEBUG
			time.Sleep(time.Second)                                                     //DEBUG
		} else {
			blocks := make([]*models.BlockConsensusMessage, 0)
			data, err := utils.ExtractResult(result)
			if err != nil {
				Service.GlobalService.Logger.Error("process - get last block from blockchain - extract data from response", zap.Error(err))
				continue
			}
			s.GlobalService.Logger.Sugar().Debugf("get last block data : %s", string(data))
			err = json.Unmarshal(data, &blocks)
			if err != nil {
				s.GlobalService.Logger.Error("handlers - process - unmarshal result of last block from blockchain collection", zap.Error(err))
				continue
			}
			block = blocks[0]
			s.GlobalService.Logger.Sugar().Debugf("Got last block from blockchain collection : %+v\n", block) //DEBUG
		}

	checkRound:

		// for testing purposes
		time.Sleep(2 * time.Second)

		s.GlobalService.Logger.Sugar().Debugf("ROUND = %d", round) //DEBUG
		time.Sleep(time.Second)                                    //DEBUG
		if round == 0 {
			// get messages with votes = 0
			transactions, _ := s.getZeroVotedTransactions(storageToken)

			s.GlobalService.Logger.Sugar().Debugf("Got transactions with votes = 0 : %+v", transactions) //DEBUG
			time.Sleep(time.Second)                                                                      //DEBUG
			consensusMsg := &models.ConsensusMessage{
				SenderAddress: s.BTCkeys.Address,
				BlockNumber:   block.Block.Number,
				Round:         round,
			}

			for _, tx := range transactions {
				err = s.validateExecuteTransactionMsg(tx, saiBtcAddress, storageToken)
				if err != nil {
					continue
				}
				s.GlobalService.Logger.Sugar().Debugf("Handling transaction : %+v", tx) //DEBUG
				time.Sleep(time.Second)                                                 //DEBUG

				consensusMsg.Messages = append(consensusMsg.Messages, tx.MessageHash)
			}

			btcResp, err := utils.SignMessage(consensusMsg, saiBtcAddress, s.BTCkeys.Private)
			if err != nil {
				s.GlobalService.Logger.Error("process - round==0 - sign consensus message", zap.Error(err))
				goto startLoop
			}

			consensusMsg.Signature = btcResp.Signature

			consensusMsgHash, err := consensusMsg.GetHash()
			if err != nil {
				s.GlobalService.Logger.Error("process - round==0 - hash consensus message", zap.Error(err))
				goto startLoop
			}

			consensusMsg.Hash = consensusMsgHash

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
			//
			msgs, err := s.getConsensusMsgForTheRound(round, storageToken)
			if err != nil {
				time.Sleep(time.Duration(s.GlobalService.Configuration["sleep"].(int)) * time.Second)
				round++

				if round <= maxRoundNumber {
					goto checkRound
				} else {
					newBlock, err := s.formAndSaveNewBlock(block, saiBtcAddress, storageToken, nil)
					if err != nil {
						continue
					}

					s.GlobalService.Logger.Sugar().Debugf(" formed new block to save: %+v\n", newBlock) //DEBUG
					time.Sleep(time.Second)                                                             //DEBUG

					err = s.broadcastMsg(newBlock.Block, saiP2Paddress)
					if err != nil {
						continue
					}
					round = 0
					goto startLoop
				}
			}

			// update votes for each transaction msg from consensus msg
			for _, msg := range msgs {
				// check if consensus message sender is from trusted validators list
				err = checkConsensusMsgSender(s.TrustedValidators, msg)
				if err != nil {
					s.GlobalService.Logger.Error("handlers - process - round != 0 - check consensus message sender", zap.Error(err))
					continue
				}
				// update votes for each tx message from consensusMsg
				for _, txMsgHash := range msg.Messages {
					err = s.updateTxMsgVotes(txMsgHash, storageToken)
					if err != nil {
						continue
					}
				}
			}
		}

		// get messages with votes>=(roundNumber*10)%
		txMsgs, err := s.getTxMsgsWithCertainNumberOfVotes(storageToken, round)
		if err != nil {
			time.Sleep(time.Duration(s.GlobalService.Configuration["sleep"].(int)) * time.Second)
			round++
			if round <= maxRoundNumber {
				goto checkRound
			} else {
				newBlock, err := s.formAndSaveNewBlock(block, saiBtcAddress, storageToken, txMsgs)
				if err != nil {
					continue
				}
				s.GlobalService.Logger.Sugar().Debugf(" formed new block to save: %+v\n", newBlock) //DEBUG
				time.Sleep(time.Second)                                                             //DEBUG

				err = s.broadcastMsg(newBlock.Block, saiP2Paddress)
				if err != nil {
					continue
				}
				round = 0
				goto startLoop
			}
		}

		newConsensusMsg := &models.ConsensusMessage{
			SenderAddress: Service.BTCkeys.Address,
			BlockNumber:   block.Block.Number,
			Round:         round,
		}

		for _, txMsg := range txMsgs {
			newConsensusMsg.Messages = append(newConsensusMsg.Messages, txMsg.MessageHash)
		}

		btcResp, err := utils.SignMessage(newConsensusMsg, saiBtcAddress, s.BTCkeys.Private)
		if err != nil {
			s.GlobalService.Logger.Error("process - round==0 - sign consensus message", zap.Error(err))
			goto startLoop
		}

		newConsensusMsg.Signature = btcResp.Signature

		newConsensusMsgHash, err := newConsensusMsg.GetHash()
		if err != nil {
			s.GlobalService.Logger.Error("process - round==0 - hash consensus message", zap.Error(err))
			goto startLoop
		}

		newConsensusMsg.Hash = newConsensusMsgHash

		err = s.broadcastMsg(newConsensusMsg, saiP2Paddress)
		if err != nil {
			continue
		}

		time.Sleep(time.Duration(s.GlobalService.Configuration["sleep"].(int)) * time.Second)
		round++

		if round <= maxRoundNumber {
			goto checkRound
		} else {
			newBlock, err := s.formAndSaveNewBlock(block, saiBtcAddress, storageToken, txMsgs)
			if err != nil {
				continue
			}

			s.GlobalService.Logger.Sugar().Debugf(" formed new block to save: %+v\n", newBlock) //DEBUG
			time.Sleep(time.Second)                                                             //DEBUG

			err = s.broadcastMsg(newBlock.Block, saiP2Paddress)
			if err != nil {
				continue
			}

			round = 0
			goto startLoop

		}
	}
}

// get last block from blockchain collection
func (s *InternalService) getLastBlockFromBlockChain(storageToken string) ([]byte, error) {
	opts := options.Find().SetSort(bson.M{"block.number": -1}).SetLimit(1)
	err, result := DB.storage.Get(blockchainCollection, bson.M{}, opts, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("handlers - process - processing - get last block", zap.Error(err))
		return nil, err
	}
	return result, nil
}

// create initial block
func (s *InternalService) createInitialBlock(address string) (block *models.BlockConsensusMessage, err error) {
	block = &models.BlockConsensusMessage{
		Votes: 0,
		Block: &models.Block{
			Number:            0,
			SenderAddress:     s.BTCkeys.Address,
			PreviousBlockHash: "",
			Messages:          make(map[string]*models.Tx),
		},
	}

	btcResp, err := utils.SignMessage(block, address, s.BTCkeys.Private)
	if err != nil {
		return nil, err
	}
	block.Block.SenderSignature = btcResp.Signature

	blockHash, err := block.Block.GetHash()
	if err != nil {
		return nil, err
	}
	block.BlockHash = blockHash
	block.Block.BlockHash = blockHash

	return block, nil

}

// get messages with votes = 0
func (s *InternalService) getZeroVotedTransactions(storageToken string) ([]*models.TransactionMessage, error) {
	err, result := DB.storage.Get("MessagesPool", bson.M{"votes": 0}, bson.M{}, storageToken)
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

	return transactions, nil
}

// validate/execute each message, update message and hash and vote for valid messages
func (s *InternalService) validateExecuteTransactionMsg(msg *models.TransactionMessage, saiBTCaddress, storageToken string) error {
	err := utils.ValidateSignature(msg, saiBTCaddress, msg.Tx.SenderAddress, msg.Tx.SenderSignature)
	if err != nil {
		s.GlobalService.Logger.Error("process - ValidateExecuteTransactionMsg - validate tx msg signature", zap.Error(err))
		return err
	}
	// dummy vm result values after executing at vm
	msg.VmResult = true
	msg.VmResponse = "vmResponse"

	msg.Votes = +1
	filter := bson.M{"message_hash": msg.MessageHash}
	update := bson.M{"votes": msg.Votes, "vm_processed": true, "vm_result": msg.VmResult, "vm_response": msg.VmResponse}
	err, _ = DB.storage.Update("MessagesPool", filter, update, storageToken)
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
func (s *InternalService) getConsensusMsgForTheRound(round int, storageToken string) ([]*models.ConsensusMessage, error) {
	err, result := DB.storage.Get("ConsensusPool", bson.M{"round": round}, bson.M{}, storageToken)
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
	data, err := json.Marshal(msg)
	if err != nil {
		s.GlobalService.Logger.Error("process - round != 0 - broadcastMsg - marshal msg", zap.Error(err))
		return err
	}

	postRequest, err := http.NewRequest("POST", saiP2Paddress, bytes.NewBuffer(data))
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

	if resp.StatusCode != 200 {
		s.GlobalService.Logger.Sugar().Errorf("process - round != 0 - broadcastMsg - send post request to host : %s wrong status code : %d", saiP2Paddress, resp.StatusCode)
		return fmt.Errorf("Wrong status code : %d", resp.StatusCode)
	}

	s.GlobalService.Logger.Sugar().Debugf("broadcasting - success, message : %+v", msg) // DEBUG
	time.Sleep(time.Second)                                                             //DEBUG
	return nil
}

// form and save new block
func (s *InternalService) formAndSaveNewBlock(previousBlock *models.BlockConsensusMessage, saiBTCaddress, storageToken string, txMsgs []*models.TransactionMessage) (*models.BlockConsensusMessage, error) {

	newBlock := &models.BlockConsensusMessage{
		Block: &models.Block{
			Number:            previousBlock.Block.Number + 1,
			PreviousBlockHash: previousBlock.BlockHash,
			SenderAddress:     s.BTCkeys.Address,
			Messages:          make(map[string]*models.Tx),
		},
	}

	for _, tx := range txMsgs {
		newBlock.Block.Messages[tx.MessageHash] = tx.Tx
	}

	btcResp, err := utils.SignMessage(newBlock, saiBTCaddress, s.BTCkeys.Private)

	if err != nil {
		s.GlobalService.Logger.Error("process - round != 0 - form and save new block - sign message", zap.Error(err))
		return nil, err

	}

	newBlock.Votes = +1
	newBlock.Block.SenderSignature = btcResp.Signature
	newBlock.Signatures = append(newBlock.Signatures, btcResp.Signature)

	blockHash, err := newBlock.Block.GetHash()
	if err != nil {
		s.GlobalService.Logger.Error("process - round != 0 - form and save new block - count hash of new block", zap.Error(err))
		return nil, err
	}
	newBlock.BlockHash = blockHash
	newBlock.Block.BlockHash = blockHash

	err, _ = DB.storage.Put(blockchainCollection, newBlock, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("process - round != 0 - form and save new block - put block to blockchain collection", zap.Error(err))
		return nil, err
	}
	return newBlock, nil
}

// update votes for transaction message
func (s *InternalService) updateTxMsgVotes(hash, storageToken string) error {
	criteria := bson.M{"message_hash": hash}
	update := bson.M{"$inc": bson.M{"votes": 1}}
	err, _ := DB.storage.Upsert("MessagesPool", criteria, update, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("handlers - process - round != 0 - get messages for specified round", zap.Error(err))
		return err
	}
	return nil
}

// get messages with certain number of votes
func (s *InternalService) getTxMsgsWithCertainNumberOfVotes(storageToken string, round int) ([]*models.TransactionMessage, error) {
	requiredVotes := float64(len(s.TrustedValidators)) / float64(round)
	filterGte := bson.M{"votes": bson.M{"$gte": requiredVotes}}
	err, result := DB.storage.Get("MessagesPool", filterGte, bson.M{}, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("handlers - process - round != 0 - get tx messages with specified votes count", zap.Float64("votes count", requiredVotes), zap.Error(err))
		return nil, err
	}

	data, err := utils.ExtractResult(result)
	if err != nil {
		Service.GlobalService.Logger.Error("process - getZeroVotedTransactions - extract data from response", zap.Error(err))
		return nil, err
	}
	txMsgs := make([]*models.TransactionMessage, 0)

	err = json.Unmarshal(data, &txMsgs)
	if err != nil {
		s.GlobalService.Logger.Error("process - round != 0 - unmarshal result of consensus messages for specified round", zap.Error(err))
		return nil, err
	}
	return txMsgs, nil
}

func (s *InternalService) getBTCkeys(fileStr, saiBTCaddress string) (*models.BtcKeys, error) {
	file, err := os.OpenFile(fileStr, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - open key btc file", zap.Error(err))
		return nil, err
	}

	data, err := ioutil.ReadFile(fileStr)
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - read key btc file", zap.Error(err))
		return nil, err
	}
	btcKeys := models.BtcKeys{}

	err = json.Unmarshal(data, &btcKeys)
	if err != nil {
		s.GlobalService.Logger.Error("get btc keys - error unmarshal from file", zap.Error(err))
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

// save test tx (for testing purposes)
func (s *InternalService) saveTestTx(saiBtcAddress, storageToken string) {
	testTxMsg := &models.TransactionMessage{
		Votes: 0,
		Tx: &models.Tx{
			SenderAddress: s.BTCkeys.Address,
			Message:       "test tx message",
		},
	}

	resp, err := utils.SignMessage(testTxMsg, saiBtcAddress, s.BTCkeys.Private)
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - sign test tx error", zap.Error(err))
	}
	testTxMsg.Tx.SenderSignature = resp.Signature

	testTxHash, err := testTxMsg.Tx.GetHash()
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - hash test tx error", zap.Error(err))
	}

	testTxMsg.Tx.MessageHash = testTxHash
	testTxMsg.MessageHash = testTxHash

	err, _ = DB.storage.Put("MessagesPool", testTxMsg, storageToken)
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - put test tx msg", zap.Error(err))
	}
}

// save test consensusMsg (for testing purposes)
func (s *InternalService) saveTestConsensusMsg(saiBtcAddress, storageToken, senderAddress string) {
	testConsensusMsg := &models.ConsensusMessage{
		SenderAddress: senderAddress,
		BlockNumber:   3,
		Round:         7,
		Messages:      []string{"0060ee497708e7d9a8428802a6651b93847dca9a0217d05ad67a5a1be7d49223"},
	}

	resp, err := utils.SignMessage(testConsensusMsg, saiBtcAddress, s.BTCkeys.Private)
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - sign test consensus error", zap.Error(err))
	}
	testConsensusMsg.Signature = resp.Signature

	testConsensusHash, err := testConsensusMsg.GetHash()
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - hash test consensus error", zap.Error(err))
	}

	testConsensusMsg.Hash = testConsensusHash

	err, _ = DB.storage.Put("ConsensusPool", testConsensusMsg, storageToken)
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - put test consensus msg", zap.Error(err))
	}
}

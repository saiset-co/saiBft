package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/bft/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	blockchainCollection = "Blockchain"
	maxRoundNumber       = 7 // maximum rounds allowed
)

// main process of blockchain
func (s *InternalService) Processing() {

	go s.listenFromSaiP2P()

	s.Logger.Sugar().Debugf("starting processing") //DEBUG

	// get btc keys
	saiBtcAddress, ok := s.GlobalService.Configuration["saiBTC_address"].(string)
	if !ok {
		s.Logger.Fatal("processing - wrong type of saiBTC address value from config")
	}
	btcKeys, err := utils.GetBtcKeys(saiBtcAddress)
	if err != nil {
		s.Logger.Fatal("processing - get btc keys", zap.Error(err))
	}

	Service.BTCkeys = btcKeys

	s.Logger.Sugar().Debugf("btc keys : %+v\n", btcKeys) //DEBUG

	storageToken, ok := s.GlobalService.Configuration["storage_token"].(string)
	if !ok {
		s.Logger.Fatal("handlers - processing - wrong type of storage token value from config")
	}

	trustedValidorsInterface, ok := s.GlobalService.Configuration["trusted_validators"].([]interface{})
	if !ok {
		s.Logger.Fatal("handlers - processing - wrong type of storage token value from config")
	}

	for _, validator := range trustedValidorsInterface {
		s.TrustedValidators = append(s.TrustedValidators, validator.(string))

	}
	s.Logger.Sugar().Debugf("got trusted validators : %v", s.TrustedValidators) //DEBUG
	// todo : insert transaction for test purposes
	testTxMsg := &models.TransactionMessage{
		Type:        models.TransactionMsgType,
		MessageHash: "d23r2fsd",
		Votes:       0,
		Tx: &models.Tx{
			Block:           1,
			VM:              "ffff",
			SenderAddress:   s.BTCkeys.Address,
			Message:         "fsdfdsf",
			SenderSignature: "dsg34",
		},
	}
	err, _ = DB.storage.Put("MessagesPool", testTxMsg, storageToken)
	if err != nil {
		s.Logger.Fatal("handlers - processing - wrong type of storage token value from config")
	}
	s.Logger.Sugar().Debugf("test tx message saved") //DEBUG

	for {
		round := 0
	startLoop:
		s.Logger.Sugar().Debugf("new round started, current round = %d", round)
		time.Sleep(5 * time.Second) //todo : for testing
		block := &models.BlockConsensusMessage{}

		// get last block from blockchain collection or create initial block
		result, err := s.getLastBlockFromBlockChain(storageToken)
		if err != nil {
			continue
		}

		// empty get response returns '{}' in storage get method -> new block should be created
		if len(result) == 2 {
			s.Logger.Sugar().Debugf("block not found, creating initial block")
			block, err = s.createInitialBlock(saiBtcAddress)
			if err != nil {
				s.Logger.Error("process - create initial block", zap.Error(err))
				continue
			}

			s.Logger.Sugar().Debugf("First block created : %+v\n", block) //DEBUG
		} else {
			data, err := utils.ExtractResult(result)
			if err != nil {
				Service.Logger.Error("process - get last block from blockchain - extract data from response", zap.Error(err))
				continue
			}
			err = json.Unmarshal(data, &block)
			if err != nil {
				s.Logger.Error("handlers - process - unmarshal result of last block from blockchain collection", zap.Error(err))
				continue
			}
			s.Logger.Sugar().Debugf("Got last block from blockchain collection : %+v\n", block) //DEBUG
		}

	checkRound:
		if round == 0 {
			// get messages with votes = 0
			transactions, err := s.getZeroVotedTransactions(storageToken)
			if err != nil {
				goto startLoop
			}

			s.Logger.Sugar().Debugf("Got transactions with votes = 0 : %+v", transactions)
			consensusMsg := &models.ConsensusMessage{
				Type:          models.ConsensusMsgType,
				SenderAddress: s.BTCkeys.Address,
				Block:         block.Block.Number,
				Round:         round,
			}

			for _, tx := range transactions {
				err = s.validateExecuteTransactionMsg(tx, storageToken)
				if err != nil {
					continue
				}
				s.Logger.Sugar().Debugf("Handling transaction : %+v", tx)

				consensusMsg.Messages = append(consensusMsg.Messages, tx.MessageHash)
			}

			s.Logger.Sugar().Debugf("signing consensus message: %+v\n", consensusMsg) //DEBUG

			btcResp, err := utils.SignMessage(consensusMsg, saiBtcAddress, s.BTCkeys.Private)
			if err != nil {
				s.Logger.Error("process - round==0 - sign consensus message", zap.Error(err))
				goto startLoop
			}

			s.Logger.Sugar().Debugf(" consensus message  signature: %+v\n", btcResp) //DEBUG

			consensusMsg.Signature = btcResp.Signature

			err = s.broadcastMsg(consensusMsg)
			if err != nil {
				s.Logger.Error("process - round==0 - broadcast consensus message", zap.Error(err))
				goto startLoop
			}

			sleepTime, ok := s.GlobalService.Configuration["sleep"].(int)
			if !ok {
				s.Logger.Fatal("wront type of sleep param in config")
			}

			time.Sleep(time.Duration(sleepTime) * time.Second)

			round++

			if round <= maxRoundNumber {
				goto checkRound
			} else {
				newBlock, err := s.formAndSaveNewBlock(block, saiBtcAddress, storageToken)
				if err != nil {
					continue
				}
				err = s.broadcastMsg(newBlock)
				if err != nil {
					continue
				}
			}

		} else {
			// get consensus messages for the round
			msgs, err := s.getConsensusMsgForTheRound(round, storageToken)
			if err != nil {
				continue
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
			continue
		}
		newConsensusMsg := &models.ConsensusMessage{
			Type:          models.ConsensusMsgType,
			SenderAddress: Service.BTCkeys.Address,
			Block:         block.Block.Number,
			Round:         round,
		}

		for _, txMsg := range txMsgs {
			newConsensusMsg.Messages = append(newConsensusMsg.Messages, txMsg.MessageHash)
		}

		btcResp, err := utils.SignMessage(newConsensusMsg, saiBtcAddress, s.BTCkeys.Private)
		if err != nil {
			s.Logger.Error("process - round==0 - sign consensus message", zap.Error(err))
			goto startLoop
		}

		newConsensusMsg.Signature = btcResp.Signature

		err = s.broadcastMsg(newConsensusMsg)
		if err != nil {
			continue
		}

		time.Sleep(s.GlobalService.Configuration["sleep"].(time.Duration) * time.Second)

		round++

		if round <= maxRoundNumber {
			goto checkRound
		} else {
			newBlock, err := s.formAndSaveNewBlock(block, saiBtcAddress, storageToken)
			if err != nil {
				continue
			}
			err = s.broadcastMsg(newBlock)
			if err != nil {
				continue
			}

		}
	}
}

// get last block from blockchain collection
func (s *InternalService) getLastBlockFromBlockChain(storageToken string) ([]byte, error) {
	opts := options.Find().SetSort(bson.M{"BlockNumber": -1}).SetLimit(1)
	err, result := DB.storage.Get(blockchainCollection, bson.M{}, opts, storageToken)
	if err != nil {
		s.Logger.Error("handlers - process - processing - get last block", zap.Error(err))
		return nil, err
	}
	return result, nil
}

// create initial block
func (s *InternalService) createInitialBlock(address string) (block *models.BlockConsensusMessage, err error) {
	block = &models.BlockConsensusMessage{
		Type:  models.BlockConsensusMsgType,
		Votes: 0,
		Block: &models.Block{
			Number:            1,
			SenderAddress:     s.BTCkeys.Address,
			PreviousBlockHash: "",
		},
	}

	btcResp, err := utils.SignMessage(block, address, s.BTCkeys.Private)
	if err != nil {
		return nil, err
	}
	block.Block.SenderSignature = btcResp.Signature

	data, err := json.Marshal(block.Block)
	if err != nil {
		s.Logger.Fatal("handlers - processing - createInitialBlock - marshal first block")
	}

	blockHash := sha256.Sum256(data)
	block.BlockHash = hex.EncodeToString(blockHash[:])

	return block, nil

}

// get messages with votes = 0
func (s *InternalService) getZeroVotedTransactions(storageToken string) ([]*models.TransactionMessage, error) {
	err, result := DB.storage.Get("MessagesPool", bson.M{"votes": 0}, bson.M{}, storageToken)
	if err != nil {
		s.Logger.Error("process - round = 0 - get messages with 0 votes", zap.Error(err))
		return nil, err
	}

	if len(result) == 2 {
		err = errors.New("no 0 voted messages found")
		s.Logger.Error("process - round = 0 - get messages with 0 votes", zap.Error(err))
		return nil, err
	}

	data, err := utils.ExtractResult(result)
	if err != nil {
		Service.Logger.Error("process - getZeroVotedTransactions - extract data from response", zap.String("data", string(result)), zap.Error(err))
		return nil, err
	}

	transactions := make([]*models.TransactionMessage, 0)

	err = json.Unmarshal(data, &transactions)
	if err != nil {
		s.Logger.Error("handlers - process - round = 0 - unmarshal result of messages with votes = 0", zap.Error(err))
		return nil, err
	}

	// todo : delete zero-voted transactions after validating and updating?
	return transactions, nil
}

// validate/execute each message, update message nad the hash and vote for valid messages
func (s *InternalService) validateExecuteTransactionMsg(msg *models.TransactionMessage, storageToken string) error {
	msg.Votes = +1
	data, err := json.Marshal(msg.Tx.Message)
	if err != nil {
		s.Logger.Error("process - round = 0 - ValidateExecuteTransactionMsg - marshal msg", zap.Error(err))
		return err
	}
	previousHash := msg.MessageHash
	blockHash := sha256.Sum256(data)
	msg.MessageHash = hex.EncodeToString(blockHash[:])
	filter := bson.M{"message_hash": previousHash}
	update := bson.M{"votes": msg.Votes, "message_hash": msg.MessageHash}
	err, _ = DB.storage.Update("MessagesPool", filter, update, storageToken)
	if err != nil {
		Service.Logger.Error("process - ValidateExecuteTransactionMsg - update transactions in storage", zap.Error(err))
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
		s.Logger.Error("process - round != 0 - get messages for specified round", zap.Error(err))
		return nil, err
	}

	data, err := utils.ExtractResult(result)
	if err != nil {
		Service.Logger.Error("process - getConsensusMsgForTheRound - extract data from response", zap.Error(err))
		return nil, err
	}

	msgs := make([]*models.ConsensusMessage, 0)

	err = json.Unmarshal(data, &msgs)
	if err != nil {
		s.Logger.Error("process - round != 0 - unmarshal result of consensus messages for specified round", zap.Error(err))
		return nil, err
	}

	return msgs, nil
}

// broadcast messages to connected nodes
func (s *InternalService) broadcastMsg(msg interface{}) error {
	s.Logger.Sugar().Debugf("broadcasting message : %+v", msg) // DEBUG
	for _, node := range s.ConnectedSaiP2pNodes {
		data, err := json.Marshal(msg)
		if err != nil {
			s.Logger.Error("process - round != 0 - broadcastMsg - marshal msg", zap.Error(err))
			return err
		}

		param := url.Values{}
		param.Add("message", string(data))

		postRequest, err := http.NewRequest("post", node.Address, strings.NewReader(param.Encode()))
		if err != nil {
			s.Logger.Error("process - round != 0 - broadcastConsensusMsg - create post request", zap.Error(err))
			return err
		}

		postRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Do(postRequest)
		if err != nil {
			s.Logger.Error("process - round != 0 - broadcastMsg - send post request", zap.Error(err))
			return err
		}

		if resp.StatusCode != 200 {
			s.Logger.Sugar().Errorf("process - round != 0 - broadcastMsg - send post request wrong status code : %d", resp.StatusCode)
			return fmt.Errorf("Wrong status code : %d", resp.StatusCode)
		}
	}
	s.Logger.Sugar().Debugf("broadcasting - success, message : %+v", msg) // DEBUG
	return nil
}

// form and save new block
func (s *InternalService) formAndSaveNewBlock(previousBlock *models.BlockConsensusMessage, saiBTCaddress, storageToken string) (*models.BlockConsensusMessage, error) {
	newBlock := &models.BlockConsensusMessage{
		Type: models.BlockConsensusMsgType,
		Block: &models.Block{
			Number:            previousBlock.Block.Number + 1,
			PreviousBlockHash: previousBlock.BlockHash,
			SenderAddress:     s.BTCkeys.Address,
		},
	}

	btcResp, err := utils.SignMessage(newBlock.Block, saiBTCaddress, s.BTCkeys.Private)
	if err != nil {
		if err != nil {
			s.Logger.Error("process - round != 0 - form and save new block - sing message", zap.Error(err))
			return nil, err
		}
	}

	newBlock.Votes = +1
	newBlock.Block.SenderSignature = btcResp.Signature

	data, err := json.Marshal(newBlock)
	if err != nil {
		s.Logger.Error("process - round != 0 - form and save new block - marshal new block", zap.Error(err))
		return nil, err
	}

	blockHash := sha256.Sum256(data)
	newBlock.BlockHash = hex.EncodeToString(blockHash[:])

	err, _ = DB.storage.Put(blockchainCollection, newBlock, storageToken)
	if err != nil {
		s.Logger.Error("process - round != 0 - form and save new block - put block to blockchain collection", zap.Error(err))
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
		s.Logger.Error("handlers - process - round != 0 - get messages for specified round", zap.Error(err))
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
		s.Logger.Error("handlers - process - round != 0 - get tx messages with specified votes count", zap.Error(err))
		return nil, err
	}

	data, err := utils.ExtractResult(result)
	if err != nil {
		Service.Logger.Error("process - getZeroVotedTransactions - extract data from response", zap.Error(err))
		return nil, err
	}
	txMsgs := make([]*models.TransactionMessage, 0)

	err = json.Unmarshal(data, &txMsgs)
	if err != nil {
		s.Logger.Error("process - round != 0 - unmarshal result of consensus messages for specified round", zap.Error(err))
		return nil, err
	}
	return txMsgs, nil
}

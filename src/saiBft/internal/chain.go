package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/bft/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// listening for messages from saiP2P

func (s *InternalService) listenFromSaiP2P(saiBTCaddress string) {

	s.Logger.Debug("saiP2P listener started")

	storageToken, ok := s.GlobalService.Configuration["storage_token"].(string)
	if !ok {
		s.Logger.Fatal("wrong type of storage_token value in config")
	}

	saiBtcAddress, ok := s.GlobalService.Configuration["saiBTC_address"].(string)
	if !ok {
		s.Logger.Fatal("wrong type of saiBTC_address value in config")
	}

	for {

		data := <-s.MsgQueue

		str, ok := data.(string)
		if !ok {
			Service.Logger.Error("listenFromSaiP2P  - wrong type of data input")
			continue
		}

		Service.Logger.Debug("got message", zap.String("message", str))
		b := []byte(str)
		m := make(map[string]interface{})
		err := json.Unmarshal(b, &m)
		if err != nil {
			Service.Logger.Error("listenFromSaiP2P  - unmarshal incoming bytes", zap.Error(err), zap.String("data", string(b)))
			continue
		}

		switch m["type"] {
		// send message to ConsensusPool collection
		case models.ConsensusMsgType:
			jsonData, err := json.Marshal(m)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - consensusMsg - marshal", zap.Error(err))
				continue
			}
			msg := models.ConsensusMessage{}

			err = json.Unmarshal(jsonData, &msg)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - consensusMsg - unmarshal", zap.Error(err))
				continue
			}
			err = msg.Validate()
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - consensusMsg - validate", zap.Error(err))
				continue
			}

			err = utils.ValidateSignature(&msg, saiBtcAddress, msg.SenderAddress, msg.Signature)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - consensusMsg - validate signature ", zap.Error(err))
				continue
			}
			err, _ = DB.storage.Put("ConsensusPool", msg, storageToken)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - consensusMsg - put to storage", zap.Error(err))
				continue
			}
			Service.Logger.Sugar().Debugf("ConsensusMsg was saved in ConsensusPool storage, msg : %+v\n", msg)
			continue

		// send message to MessagesPool collection
		case models.TransactionMsgType:
			jsonData, err := json.Marshal(m)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - transactionMsg - marshal", zap.Error(err))
				continue
			}
			msg := models.TransactionMessage{}

			err = json.Unmarshal(jsonData, &msg)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - transactionMsg - unmarshal", zap.Error(err))
				continue
			}
			err = msg.Validate()
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - transactionMsg - validate", zap.Error(err))
				continue
			}

			err = utils.ValidateSignature(&msg, saiBtcAddress, msg.Tx.SenderAddress, msg.Tx.SenderSignature)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - consensusMsg - validate signature ", zap.Error(err))
				continue
			}

			err, _ = DB.storage.Put("MessagesPool", msg, storageToken)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - transactionMsg - put to storage", zap.Error(err))
				continue
			}
			Service.Logger.Sugar().Debugf("TransactionMsg was saved in MessagesPool storage, msg : %+v\n", msg)
			continue

		// handle BlockConsensusMsg
		case models.BlockConsensusMsgType:
			jsonData, err := json.Marshal(m)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - blockConsensusMsg - marshal", zap.Error(err))
				continue
			}
			msg := models.BlockConsensusMessage{}

			err = json.Unmarshal(jsonData, &msg)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - blockConsensusMsg - unmarshal", zap.Error(err))
				continue
			}
			err = msg.Validate()
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - blockConsensusMsg - validate", zap.Error(err))
				continue
			}

			err = utils.ValidateSignature(&msg, saiBtcAddress, msg.Block.SenderAddress, msg.Block.SenderSignature)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - blockConsensusMsg - validate signature ", zap.Error(err))
				continue
			}

			err = s.handleBlockConsensusMsg(saiBTCaddress, storageToken, &msg)
			if err != nil {
				continue
			}
		default:
			Service.Logger.Error("listenFromSaiP2P - blockConsensusMsg - wrong message type to handle")
			continue
		}
	}
}

// handle BlockConsensusMsg
func (s *InternalService) handleBlockConsensusMsg(saiBTCaddress, storageToken string, msg *models.BlockConsensusMessage) error {
	isValid := s.validateBlockConsensusMsg(msg)

	if !isValid {
		err := errors.New("Provided BlockConsensusMsg is invalid")
		s.Logger.Error("handleBlockConsensusMsg - validate message and sender", zap.Error(err))
		return err
	}

	err, result := DB.storage.Get(blockchainCollection, bson.M{"block.number": msg.Block.Number}, bson.M{}, storageToken)
	if err != nil {
		s.Logger.Error("handleBlockConsensusMsg - get block N ", zap.Error(err))
		return err
	}

	s.Logger.Sugar().Debugf("got block consensus : %s\n", result)

	if len(result) == 2 {
		err = fmt.Errorf("block with number = %d was not found", msg.Block.Number)
		s.Logger.Error("handleBlockConsensusMsg - get block N", zap.Error(err))
		return err
	}

	data, err := utils.ExtractResult(result)
	if err != nil {
		s.Logger.Error("handleBlockConsensusMsg - extract data from response", zap.Error(err))
		return err
	}

	block := models.BlockConsensusMessage{}

	err = json.Unmarshal(data, &block)
	if err != nil {
		s.Logger.Error("handleBlockConsensusMsg - unmarshal block", zap.Error(err))
		return err
	}

	if block.BlockHash == msg.BlockHash {

		// resp, err := utils.SignMessage(msg, saiBTCaddress, s.BTCkeys.Private)
		// if err != nil {
		// 	s.Logger.Error("handleBlockConsensusMsg - sign message", zap.Error(err))
		// 	return err
		// }
		// todo : which signature to add?
		block.Signatures = append(block.Signatures, block.Block.SenderSignature)
		block.Votes++
		filter := bson.M{"number": block.Block.Number}
		update := bson.M{"votes": block.Votes, "voted_singatures": block.Signatures}
		err, _ = DB.storage.Update(blockchainCollection, filter, update, storageToken)
		if err != nil {
			s.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - update votes in storage", zap.Error(err))
			return err
		}
		s.Logger.Sugar().Debugf("votes was updated in blockchain storage for block : %+v\n", block)
		return nil
	} else {
		err, result := DB.storage.Get("BlockCandidates", bson.M{"hash": block.BlockHash}, bson.M{}, storageToken)
		if err != nil {
			s.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - get block candidate by hash", zap.Error(err))
			return err
		}

		// empty get response returns '{}' in storage get method
		if len(result) == 2 {
			// insert block to BlockCandidates collection
			err, _ := DB.storage.Put("BlockCandidates", block, storageToken)
			if err != nil {
				s.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - insert block to BlockCandidates collection", zap.Error(err))
				return err
			}

			s.Logger.Sugar().Debugf("block candidate was inserted to blockCandidates collection, blockCandidate : %+v\n", block)
			return nil
		}

		data, err = utils.ExtractResult(result)
		if err != nil {
			s.Logger.Error("handleBlockConsensusMsg - extract data from response", zap.Error(err))
			return err
		}

		blockCandidate := models.BlockConsensusMessage{}

		err = json.Unmarshal(data, &blockCandidate)
		if err != nil {
			s.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - unmarshal blockCandidate", zap.Error(err))
			return err
		}

		s.Logger.Sugar().Debugf("got block candidate : %+v\n", blockCandidate)

		blockCandidate.Votes++
		// todo : which signature to add?
		blockCandidate.Signatures = append(blockCandidate.Signatures, blockCandidate.Block.SenderSignature)
		if blockCandidate.Votes < block.Votes {
			resultBlocks, err := s.sendDirectGetBlockMsg(block.Block.Number)
			if err != nil {
				s.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - sendDirectGetBlockMessage", zap.Error(err))
				return err
			}

			err, _ = DB.storage.Put(blockchainCollection, resultBlocks, storageToken)
			if err != nil {
				return err
			}
			s.Logger.Sugar().Debugf("blockCandidate was saved in blockchain collection, msg : %+v\n", blockCandidate)
			return nil

		} else {
			return nil
		}

	}
}

func (s *InternalService) sendDirectGetBlockMsg(lastBlockNumber int) (resultBlocks []*models.BlockConsensusMessage, err error) {
	// temp map for comparing missed blocks, which got from connected saiP2p nodes
	tempMap := make(map[*models.BlockConsensusMessage]int)

	for _, node := range s.ConnectedSaiP2pNodes {
		blocks, err := utils.SendDirectGetBlockMsg(node.Address, lastBlockNumber)
		if err != nil {
			s.Logger.Error("chain - send direct get block message", zap.String("node", node.Address), zap.Error(err))
			continue
		}
		for _, b := range blocks {
			tempMap[b]++
		}
	}
	resultBlocks = make([]*models.BlockConsensusMessage, 0)

	for i := 1; i <= lastBlockNumber; i++ {
		sliceToSort := make([]*models.BlockConsensusMessage, 0)
		for k, v := range tempMap {
			if k.Block.Number == i {
				sliceToSort = append(sliceToSort, &models.BlockConsensusMessage{
					Count: v,
					Block: k.Block,
				})
			}
		}
		sort.Slice(sliceToSort, func(i, j int) bool {
			return sliceToSort[i].Count > sliceToSort[j].Count
		})
		if len(sliceToSort) == 1 {
			resultBlocks = append(resultBlocks, sliceToSort[0])
		} else {
			if sliceToSort[0].Count != sliceToSort[1].Count {
				resultBlocks = append(resultBlocks, sliceToSort[0])
			}
		}
	}
	return resultBlocks, nil
}

// validate block consensus message
func (s *InternalService) validateBlockConsensusMsg(msg *models.BlockConsensusMessage) bool {
	// trustedValidators, ok := s.GlobalService.Configuration["trusted_validators"].([]string)
	// if !ok {
	// 	s.Logger.Fatal("wrong type of trusted_validators in config")
	// }
	for _, validator := range s.TrustedValidators {
		if validator == msg.Block.SenderAddress {
			return true
		}
	}

	// todo : dummy response for test
	return false
}

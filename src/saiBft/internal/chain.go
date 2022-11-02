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

	s.GlobalService.Logger.Debug("saiP2P listener started")

	storageToken, ok := s.GlobalService.Configuration["storage_token"].(string)
	if !ok {
		s.GlobalService.Logger.Fatal("wrong type of storage_token value in config")
	}

	saiBtcAddress, ok := s.GlobalService.Configuration["saiBTC_address"].(string)
	if !ok {
		s.GlobalService.Logger.Fatal("wrong type of saiBTC_address value in config")
	}

	saiP2Paddress, ok := s.GlobalService.Configuration["saiP2P_address"].(string)
	if !ok {
		s.GlobalService.Logger.Fatal("processing - wrong type of saiP2P address value from config")
	}
	btckeys, err := s.getBTCkeys("btc_keys.json", saiBtcAddress)
	if err != nil {
		Service.GlobalService.Logger.Fatal("listenFromSaiP2P  - handle tx msg - get btc keys", zap.Error(err))
	}
	s.BTCkeys = btckeys

	for {
		data := <-s.MsgQueue
		switch data.(type) {
		case *models.Tx:
			txMsg := data.(*models.Tx)
			Service.GlobalService.Logger.Sugar().Debugf("got tx message : %+v", txMsg)

			//todo : tx msg in bytes or struct, not string

			msg := &models.TransactionMessage{
				Tx:          txMsg,
				MessageHash: txMsg.MessageHash,
			}

			err = msg.Validate()
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - transactionMsg - validate", zap.Error(err))
				continue
			}

			err = utils.ValidateSignature(msg, saiBtcAddress, msg.Tx.SenderAddress, msg.Tx.SenderSignature)
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - consensusMsg - validate signature ", zap.Error(err))
				continue
			}

			err = s.broadcastMsg(msg.Tx, saiP2Paddress)
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P  - handle tx msg - broadcast tx", zap.Error(err))
				continue
			}

			err, _ = DB.storage.Put("MessagesPool", msg, storageToken)
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - transactionMsg - put to storage", zap.Error(err))
				continue
			}
			Service.GlobalService.Logger.Sugar().Debugf("TransactionMsg was saved in MessagesPool storage, msg : %+v\n", msg)
			s.MsgQueue <- struct{}{}

		case *models.ConsensusMessage:
			msg := data.(*models.ConsensusMessage)
			err = msg.Validate()
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - consensusMsg - validate", zap.Error(err))
				continue
			}

			err = utils.ValidateSignature(&msg, saiBtcAddress, msg.SenderAddress, msg.Signature)
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - consensusMsg - validate signature ", zap.Error(err))
				continue
			}
			err, _ = DB.storage.Put("ConsensusPool", msg, storageToken)
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - consensusMsg - put to storage", zap.Error(err))
				continue
			}
			Service.GlobalService.Logger.Sugar().Debugf("ConsensusMsg was saved in ConsensusPool storage, msg : %+v\n", msg)
			continue

		case *models.Block:

		}
	}
}

// handle BlockConsensusMsg
func (s *InternalService) handleBlockConsensusMsg(saiBTCaddress, storageToken string, msg *models.BlockConsensusMessage) error {
	isValid := s.validateBlockConsensusMsg(msg)

	if !isValid {
		err := errors.New("Provided BlockConsensusMsg is invalid")
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - validate message and sender", zap.Error(err))
		return err
	}

	// todo : if there is no such block? -
	err, result := DB.storage.Get(blockchainCollection, bson.M{"block.number": msg.Block.Number}, bson.M{}, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - get block N ", zap.Error(err))
		return err
	}

	if len(result) == 2 {
		err = fmt.Errorf("block with number = %d was not found", msg.Block.Number)
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - get block N", zap.Error(err))
		return err
	}

	s.GlobalService.Logger.Sugar().Debugf("got block consensus : %s\n", result)

	data, err := utils.ExtractResult(result)
	if err != nil {
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - extract data from response", zap.Error(err))
		return err
	}

	block := models.BlockConsensusMessage{}

	err = json.Unmarshal(data, &block)
	if err != nil {
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - unmarshal block", zap.Error(err))
		return err
	}

	if block.BlockHash == msg.BlockHash {
		block.Signatures = append(block.Signatures, msg.Block.SenderSignature)
		block.Votes++

		filter := bson.M{"number": block.Block.Number}
		update := bson.M{"votes": block.Votes, "voted_singatures": block.Signatures}
		err, _ = DB.storage.Update(blockchainCollection, filter, update, storageToken)
		if err != nil {
			s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - update votes in storage", zap.Error(err))
			return err
		}

		s.GlobalService.Logger.Sugar().Debugf("votes was updated in blockchain storage for block : %+v\n", block)
		return nil
	} else {
		// get and process block candidate
		blockCandidate, err := s.getBlockCandidate(msg, storageToken)
		if err != nil {
			s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash != msgBlockHash - getBlockCandidates", zap.Error(err))
			return err
		}

		s.GlobalService.Logger.Sugar().Debugf("got block candidate : %+v\n", blockCandidate)

		blockCandidate.Votes++
		blockCandidate.Signatures = append(blockCandidate.Signatures, msg.Block.SenderSignature)

		if blockCandidate.Votes < block.Votes {
			resultBlocks, err := s.sendDirectGetBlockMsg(block.Block.Number)
			if err != nil {
				s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - sendDirectGetBlockMessage", zap.Error(err))
				return err
			}

			err, _ = DB.storage.Put(blockchainCollection, resultBlocks, storageToken)
			if err != nil {
				return err
			}
			s.GlobalService.Logger.Sugar().Debugf("blockCandidate was saved in blockchain collection, msg : %+v\n", blockCandidate)
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
			s.GlobalService.Logger.Error("chain - send direct get block message", zap.String("node", node.Address), zap.Error(err))
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
	// 	s.GlobalService.Logger.Fatal("wrong type of trusted_validators in config")
	// }
	for _, validator := range s.TrustedValidators {
		if validator == msg.Block.SenderAddress {
			return true
		}
	}

	// todo : dummy response for test
	return false
}

// get block candidate with the block hash
// add vote if exists, insert blockCandidate, if not exists
func (s *InternalService) getBlockCandidate(msg *models.BlockConsensusMessage, storageToken string) (*models.BlockConsensusMessage, error) {
	err, result := DB.storage.Get("BlockCandidates", bson.M{"hash": msg.Block.BlockHash}, bson.M{}, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash != msgBlockHash - get block candidate by msg block hash", zap.Error(err))
		return nil, err
	}
	// empty get response returns '{}' in storage get method
	if len(result) == 2 {
		err, _ := DB.storage.Put("BlockCandidates", msg, storageToken)
		if err != nil {
			s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - insert block to BlockCandidates collection", zap.Error(err))
			return nil, err
		}
		s.GlobalService.Logger.Sugar().Debugf("block candidate was inserted to blockCandidates collection, blockCandidate : %+v\n", msg)
		return msg, nil
	}
	data, err := utils.ExtractResult(result)
	if err != nil {
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - extract data from response", zap.Error(err))
		return nil, err
	}
	blockCandidate := models.BlockConsensusMessage{}

	err = json.Unmarshal(data, &blockCandidate)
	if err != nil {
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - unmarshal blockCandidate", zap.Error(err))
		return nil, err
	}
	return &blockCandidate, nil
}

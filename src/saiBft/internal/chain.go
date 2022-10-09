package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/bft/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// listening for messages from saiP2P
func (s *InternalService) listenFromSaiP2P() {
	host, ok := s.GlobalService.Configuration["socket_host"].(string)
	if !ok {
		s.Logger.Fatal("wrong type of socket_host value in config")
	}

	port, ok := s.GlobalService.Configuration["socket_port"].(int)
	if !ok {
		s.Logger.Fatal("wrong type of socket_port value in config")
	}

	socketServer, err := net.Listen("tcp", host+":"+fmt.Sprint(port))
	if err != nil {
		s.Logger.Fatal("error creating socket connection", zap.Error(err))
	}

	defer socketServer.Close()

	s.Logger.Sugar().Debug("chain (listening saiP2p socket) started on %s:%s", host, strconv.Itoa(port))

	storageToken, ok := s.GlobalService.Configuration["storage_token"].(string)
	if !ok {
		s.Logger.Fatal("wrong type of storage_token value in config")
	}

	saiBtcAddress, ok := s.GlobalService.Configuration["saiBTC_address"].(string)
	if !ok {
		s.Logger.Fatal("wrong type of saiBTC_address value in config")
	}

	for {
		msg := make(map[string]interface{})
		conn, err := socketServer.Accept()
		if err != nil {
			s.Logger.Error("listenFromSaiP2P - accept from connection", zap.Error(err))
			continue
		}
		var buf bytes.Buffer
		io.Copy(&buf, conn)

		err = json.Unmarshal(buf.Bytes(), &msg)
		if err != nil {
			Service.Logger.Error("listenFromSaiP2P  - unmarshal incoming bytes", zap.Error(err))
			continue
		}

		switch msg["type"] {
		// send message to ConsensusPool collection
		case models.ConsensusMsgType:
			msg := models.ConsensusMessage{}
			err = json.Unmarshal(buf.Bytes(), &msg)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - consensusMsg - unmarshal", zap.Error(err))
				continue
			}
			err = msg.Validate()
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - consensusMsg - validate", zap.Error(err))
				continue
			}

			err = utils.ValidateSignature(msg, saiBtcAddress, msg.SenderAddress, msg.Signature)
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
			msg := models.TransactionMessage{}
			err = json.Unmarshal(buf.Bytes(), &msg)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - transactionMsg - unmarshal", zap.Error(err))
				continue
			}
			err = msg.Validate()
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - transactionMsg - validate", zap.Error(err))
				continue
			}

			err = utils.ValidateSignature(msg, saiBtcAddress, msg.Tx.SenderAddress, msg.Tx.SenderSignature)
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
			msg := models.BlockConsensusMessage{}
			err = json.Unmarshal(buf.Bytes(), &msg)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - blockConsensusMsg - unmarshal", zap.Error(err))
				continue
			}
			err = msg.Validate()
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - blockConsensusMsg - validate", zap.Error(err))
				continue
			}

			err = utils.ValidateSignature(msg, saiBtcAddress, msg.SenderAddress, msg.SenderSignature)
			if err != nil {
				Service.Logger.Error("listenFromSaiP2P - consensusMsg - validate signature ", zap.Error(err))
				continue
			}

			err := s.handleBlockConsensusMsg(storageToken, &msg)
			if err != nil {
				continue
			}
		}
	}
}

// handle BlockConsensusMsg
func (s *InternalService) handleBlockConsensusMsg(storageToken string, msg *models.BlockConsensusMessage) error {
	isValid := s.validateBlockConsensusMsg(msg)

	if !isValid {
		err := errors.New("Provided BlockConsensusMsg is invalid")
		s.Logger.Error("handleBlockConsensusMsg - validate message and sender", zap.Error(err))
		return err
	}

	err, result := DB.storage.Get(blockchainCollection, bson.M{"block_number": msg.BlockNumber}, bson.M{}, storageToken)
	if err != nil {
		s.Logger.Error("handleBlockConsensusMsg - get block N ", zap.Error(err))
		return err
	}

	s.Logger.Sugar().Debugf("got block consensus : %s\n", result)

	if len(result) == 2 {
		err = fmt.Errorf("block with number = %d was not found", msg.BlockNumber)
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
		block.Votes++
		filter := bson.M{"block_number": block.BlockNumber}
		update := bson.M{"votes": block.Votes}
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
		if blockCandidate.Votes < block.Votes {
			resultBlocks, err := s.sendDirectGetBlockMsg(block.BlockNumber)
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

// todo : на приконечненные ноды отправляем сообщение
/// сформируейте все блоки от такого до такого, которые у меня пропущены
// how to find out, which blocks i have missed
//  simple : get block from [0:N]  from another connected votes
// получение блоков сортировка по номеру, ограничение по количеству
// broadcast messages to connected nodes
func (s *InternalService) sendDirectGetBlockMsg(blockNumber int) (resultBlocks []*models.BlockConsensusMessage, err error) {

	// temp map for comparing missed blocks, which got from connected saiP2p nodes
	tempMap := make(map[int]*models.BlockConsensusMessage)

	for _, node := range s.ConnectedSaiP2pNodes {
		blocks, err := utils.SendDirectGetBlockMsg(node.Address, blockNumber)
		if err != nil {
			s.Logger.Error("chain - send direct get block message", zap.String("node", node.Address), zap.Error(err))
			continue
		}

		// todo : ask about algorhytm to compare missed blocks from connected nodes
		for i, block := range blocks {
			comparedBlock, ok := tempMap[i]
			if !ok {
				tempMap[i] = block
			} else {
				if comparedBlock.BlockHash == block.BlockHash {
					continue
				} else {
					if block.Votes > comparedBlock.Votes {
						tempMap[i] = block
					} else {
						continue
					}
				}
			}
		}
	}

	for _, block := range tempMap {
		resultBlocks = append(resultBlocks, block)
	}

	return resultBlocks, nil
}

// validate block consensus message
func (s *InternalService) validateBlockConsensusMsg(msg *models.BlockConsensusMessage) bool {
	trustedValidators, ok := s.GlobalService.Configuration["trusted_validators"].([]string)
	if !ok {
		s.Logger.Fatal("wrong type of trusted_validators in config")
	}
	for _, validator := range trustedValidators {
		if validator == msg.SenderAddress {
			return true
		}
	}
	return false
}

package internal

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"

	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/bft/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// listening for messages from saiP2P
func (s *InternalService) listenFromSaiP2P(saiBTCaddress string) {
	s.GlobalService.Logger.Debug("saiP2P listener started") // DEBUG

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

	saiP2pProxyAddress, ok := s.GlobalService.Configuration["saiProxy_address"].(string)
	if !ok {
		s.GlobalService.Logger.Fatal("processing - wrong type of saiP2pProxy address value from config")
	}

	for {
		data := <-s.MsgQueue
		s.GlobalService.Logger.Debug("chain - got data", zap.Any("data", data)) // DEBUG
		switch data.(type) {
		case *models.Tx:
			// skip if state is not initialized
			if !s.SkipInitializating && !s.IsInitialized {
				continue
			}
			txMsg := data.(*models.Tx)
			Service.GlobalService.Logger.Sugar().Debugf("chain - got tx message : %+v", txMsg) //DEBUG

			msg := &models.TransactionMessage{
				Tx:          txMsg,
				MessageHash: txMsg.MessageHash,
			}
			err := msg.Validate()
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - transactionMsg - validate", zap.Error(err))
				continue
			}
			err = utils.ValidateSignature(msg, saiBtcAddress, msg.Tx.SenderAddress, msg.Tx.SenderSignature)
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - tx msg - validate signature ", zap.Error(err))
				continue
			}

			err, result := s.Storage.Get("MessagesPool", msg, bson.M{}, storageToken)
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - transactionMsg - get from storage", zap.Error(err))
				continue
			}

			if len(result) > 2 {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - transactionMsg - we have sent this message", zap.Error(err))
				continue
			}

			err, _ = s.Storage.Put("MessagesPool", msg, storageToken)
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - transactionMsg - put to storage", zap.Error(err))
				continue
			}

			Service.GlobalService.Logger.Sugar().Debugf("TransactionMsg was saved in MessagesPool storage, msg : %+v\n", msg)

		case *models.ConsensusMessage:
			// skip if state is not initialized
			if !s.SkipInitializating && !s.IsInitialized {
				continue
			}
			msg := data.(*models.ConsensusMessage)
			Service.GlobalService.Logger.Sugar().Debugf("chain - got consensus message : %+v", msg) //DEBUG
			err := msg.Validate()
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - consensusMsg - validate", zap.Error(err))
				continue
			}
			err = utils.ValidateSignature(msg, saiBtcAddress, msg.SenderAddress, msg.Signature)
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - consensusMsg - validate signature ", zap.Error(err))
				continue
			}
			err, _ = s.Storage.Put("ConsensusPool", msg, storageToken)
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - consensusMsg - put to storage", zap.Error(err))
				continue
			}
			Service.GlobalService.Logger.Sugar().Debugf("ConsensusMsg was saved in ConsensusPool storage, msg : %+v\n", msg)
			continue

		case *models.BlockConsensusMessage:
			msg := data.(*models.BlockConsensusMessage)
			Service.GlobalService.Logger.Sugar().Debugf("chain - got block consensus message : %+v", msg) //DEBUG
			err := msg.Validate()
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - consensusMsg - validate", zap.Error(err))
				continue
			}
			err = utils.ValidateSignature(msg, saiBtcAddress, msg.Block.SenderAddress, msg.Block.SenderSignature)
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - consensusMsg - validate signature ", zap.Error(err))
				continue
			}

			if !s.SkipInitializating && !s.IsInitialized {
				err, _ = s.Storage.Put(blockchainCol, msg, storageToken)
				if err != nil {
					Service.GlobalService.Logger.Error("listenFromSaiP2P - initial block consensus msg - put to storage", zap.Error(err))
				}
				s.InitialSignalCh <- struct{}{}
				continue
			}

			err = s.handleBlockConsensusMsg(saiBtcAddress, saiP2pProxyAddress, storageToken, msg, saiP2Paddress)
			if err != nil {
				Service.GlobalService.Logger.Error("listenFromSaiP2P - block consensus msg - put to storage", zap.Error(err))
				continue
			}
		default:
			Service.GlobalService.Logger.Error("listenFromSaiP2P - got wrong msg type", zap.Any("type", reflect.TypeOf(data)))
		}
	}
}

// handle BlockConsensusMsg
func (s *InternalService) handleBlockConsensusMsg(saiBTCaddress, saiP2pProxyAddress, storageToken string, msg *models.BlockConsensusMessage, saiP2pAddress string) error {
	isValid := s.validateBlockConsensusMsg(msg)
	if !isValid {
		err := errors.New("Provided BlockConsensusMsg is invalid")
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - validate message and sender", zap.Error(err))
		return err
	}

	// check if we are not at 7 round in process
	if s.Round7State {
		s.GlobalService.Logger.Debug("chain - 7 round state detected")
		err, _ := s.Storage.Put("BlockCandidates", msg, storageToken)
		if err != nil {
			s.GlobalService.Logger.Error("handleBlockConsensusMsg - 7 round state - insert block to BlockCandidates collection", zap.Error(err))
			return err
		}
		s.GlobalService.Logger.Sugar().Debugf("block candidate was inserted to blockCandidates collection (7 round state), blockCandidate : %+v\n", msg) // DEBUG
		return nil
	}

	// Get Block N
	err, result := s.Storage.Get(blockchainCol, bson.M{"block.number": msg.Block.Number}, bson.M{}, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - get block N ", zap.Error(err))
		return err
	}

	// if there is no such block - go futher (compare block hash)
	if len(result) == 2 {
		err, _ := s.Storage.Put(blockchainCol, msg, storageToken)
		if err != nil {
			s.GlobalService.Logger.Error("handleBlockConsensusMsg - 7 round state - insert block to BlockCandidates collection", zap.Error(err))
			return err
		}
		s.GoToStartLoopCh <- struct{}{}

		err = s.GetMissedBlocks(msg.Block.Number, storageToken)
		if err != nil {
			s.GlobalService.Logger.Error("handleBlockConsensusMsg - GetMissedBlocks", zap.Error(err))
			return err
		}
		//todo : not ready
		//	return s.handleBlockCandidate(msg, saiP2pProxyAddress, saiP2pAddress, storageToken)
	}

	s.GlobalService.Logger.Sugar().Debugf("got block consensus : %s\n", result)

	data, err := utils.ExtractResult(result)
	if err != nil {
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - extract data from response", zap.Error(err))
		return err
	}

	blocks := []models.BlockConsensusMessage{}
	err = json.Unmarshal(data, &blocks)
	if err != nil {
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - unmarshal block", zap.Error(err))
		return err
	}

	block := blocks[0]

	if block.BlockHash == msg.BlockHash {
		err := s.addVotesToBlock(&block, msg, storageToken)
		if err != nil {
			s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - add votes to block", zap.Error(err))
			return err
		}
		s.GlobalService.Logger.Sugar().Debugf("votes was updated in blockchain storage for block : %+v\n", block)
		return nil
	} else {
		return s.handleBlockCandidate(msg, saiP2pProxyAddress, saiP2pAddress, storageToken)
	}
}

//todo : old func
// Get missed blocks from connected nodes & compare received blocks
func (s *InternalService) sendDirectGetBlockMsg(lastBlockNumber int, saiP2pProxyAddress string, saiP2pAddress string) (resultBlocks []*models.BlockConsensusMessage, err error) {
	// temp map for comparing missed blocks, which got from connected saiP2p nodes
	//tempMap := make(map[*models.BlockConsensusMessage]int)

	addresses, err := utils.GetConnectedNodesAddresses(saiP2pProxyAddress, s.GlobalService.Configuration["nodes_blacklist"].([]string))
	if err != nil {
		s.GlobalService.Logger.Error("chain - handleBlockConsensusMsg - sendDirectGetBlockMsg", zap.Error(err))
		return nil, err
	}

	for _, node := range addresses {
		err := utils.SendDirectGetBlockMsg(node, &models.SyncRequest{}, saiP2pAddress)
		if err != nil {
			s.GlobalService.Logger.Error("chain - send direct get block message", zap.String("node", node), zap.Error(err))
			continue
		}
		// for _, b := range blocks {
		// 	tempMap[b]++
		// }
	}
	resultBlocks = make([]*models.BlockConsensusMessage, 0)

	// for i := 1; i <= lastBlockNumber; i++ {
	// 	sliceToSort := make([]*models.BlockConsensusMessage, 0)
	// 	for k, v := range tempMap {
	// 		if k.Block.Number == i {
	// 			sliceToSort = append(sliceToSort, &models.BlockConsensusMessage{
	// 				Count: v,
	// 				Block: k.Block,
	// 			})
	// 		}
	// 	}
	// 	sort.Slice(sliceToSort, func(i, j int) bool {
	// 		return sliceToSort[i].Count > sliceToSort[j].Count
	// 	})
	// 	if len(sliceToSort) == 1 {
	// 		resultBlocks = append(resultBlocks, sliceToSort[0])
	// 	} else {
	// 		if sliceToSort[0].Count != sliceToSort[1].Count {
	// 			resultBlocks = append(resultBlocks, sliceToSort[0])
	// 		}
	// 	}
	// }
	return resultBlocks, nil
}

// validate block consensus message
func (s *InternalService) validateBlockConsensusMsg(msg *models.BlockConsensusMessage) bool {
	for _, validator := range s.TrustedValidators {
		if validator == msg.Block.SenderAddress {
			return true
		}
	}

	return false
}

// get block candidate with the block hash, add vote if exists
// insert blockCandidate, if not exists
func (s *InternalService) getBlockCandidate(blockHash string, storageToken string) (*models.BlockConsensusMessage, error) {
	err, result := s.Storage.Get("BlockCandidates", bson.M{"hash": blockHash}, bson.M{}, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash != msgBlockHash - get block candidate by msg block hash", zap.Error(err))
		return nil, err
	}

	if len(result) == 2 {
		return nil, nil
	}

	blockCandidates := make([]models.BlockConsensusMessage, 0)
	err = json.Unmarshal(result, &blockCandidates)
	if err != nil {
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockCandidateHash = msgBlockHash - unmarshal", zap.Error(err))
		return nil, err
	}

	blockCandidate := blockCandidates[0]
	s.GlobalService.Logger.Sugar().Debugf("got block candidate : %+v\n", blockCandidate) //DEBUG
	return &blockCandidate, nil
}

// add vote to block N
func (s *InternalService) addVotesToBlock(block, msg *models.BlockConsensusMessage, storageToken string) error {
	block.Signatures = append(block.Signatures, msg.Block.SenderSignature)
	block.Votes++
	filter := bson.M{"block.number": block.Block.Number}
	update := bson.M{"votes": block.Votes, "voted_signatures": block.Signatures}
	err, _ := s.Storage.Update(blockchainCol, filter, update, storageToken)
	return err
}

// update blockchain
// 1. get missed blocks from connected nodes
// 2. put missed and chosen blocks to blockchain collection
func (s *InternalService) updateBlockchain(msg, blockCandidate *models.BlockConsensusMessage, storageToken, saiP2pProxyAddress, saiP2pAddress string) error {
	//resultBlocks, err := s.sendDirectGetBlockMsg(msg.Block.Number, saiP2pProxyAddress, saiP2pAddress)
	resultBlocks, err := s.sendDirectGetBlockMsg(msg.Block.Number, saiP2pProxyAddress, saiP2pAddress)
	if err != nil {
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - sendDirectGetBlockMessage", zap.Error(err))
		return err
	}

	err, _ = s.Storage.Put(blockchainCol, resultBlocks, storageToken)
	if err != nil {
		return err
	}
	s.GlobalService.Logger.Sugar().Debugf("blockCandidate was saved in blockchain collection, msg : %+v\n", blockCandidate)
	return nil
}

// Block candidate logic
// 1. Get block candidate from db
//2. update blockchain if blockCandidate votes < incoming msg.Votes
func (s *InternalService) handleBlockCandidate(msg *models.BlockConsensusMessage, saiP2pProxyAddress, saiP2pAddress, storageToken string) error {
	blockCandidate, err := s.getBlockCandidate(msg.BlockHash, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash != msgBlockHash - get block candidate by msg block hash", zap.Error(err))
		return err
	}

	// empty get response returns '{}' in storage get method
	if blockCandidate == nil {
		if float64(msg.Votes) > math.Ceil(float64(len(s.TrustedValidators))*7/10) {
			err, _ := s.Storage.Put("Blockchain", msg, storageToken)
			if err != nil {
				s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - insert block to blockchain collection", zap.Error(err))
				return err
			}
			s.GlobalService.Logger.Sugar().Debugf("block candidate was inserted to blockchain collection, blockCandidate : %+v\n", msg) // DEBUG
			err = s.GetMissedBlocks(msg.Block.Number, storageToken)
		} else {
			err, _ := s.Storage.Put("BlockCandidates", msg, storageToken)
			if err != nil {
				s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - insert block to BlockCandidates collection", zap.Error(err))
				return err
			}
			s.GlobalService.Logger.Sugar().Debugf("block candidate was inserted to blockCandidates collection, blockCandidate : %+v\n", msg) // DEBUG
			return nil
		}

	}

	blockCandidate.Votes++
	blockCandidate.Signatures = append(blockCandidate.Signatures, msg.Block.SenderSignature)
	if blockCandidate.Votes > msg.Votes {
		err, _ := s.Storage.Put("Blockchain", msg, storageToken)
		if err != nil {
			s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - insert block to BlockCandidates collection", zap.Error(err))
			return err
		}

		s.GlobalService.Logger.Sugar().Debugf("block candidate was inserted to blockchain collection, blockCandidate : %+v\n", msg) // DEBUG

		err = s.updateBlockchain(msg, blockCandidate, saiP2pProxyAddress, storageToken, saiP2pAddress)
		if err != nil {
			s.GlobalService.Logger.Error("handleBlockConsensusMsg - blockHash = msgBlockHash - update blockchain", zap.Error(err))
			return err
		}
	}

	return nil

}

// get blockchain missed blocks
// send direct get block message to connected nodes
// get blocks from n to current
func (s *InternalService) GetMissedBlocks(blockNumber int, storageToken string) error {
	saiP2Paddress, ok := s.GlobalService.Configuration["saiP2P_address"].(string)
	if !ok {
		s.GlobalService.Logger.Fatal("chain - handleBlockCandidate - GetBlockchainMissedBlocks - create post request- wrong type of saiP2P address value from config")
	}
	blacklistIface, ok := s.GlobalService.Configuration["nodes_blacklist"].([]interface{})
	if !ok {
		s.GlobalService.Logger.Fatal("chain - handleBlockCandidate - GetBlockchainMissedBlocks - create post request- wrong type of nodes_blacklist  value from config", zap.String("detected type", reflect.ValueOf(s.GlobalService.Configuration["nodes_blacklist"]).String()))
	}

	blacklist := make([]string, 0)
	for _, b := range blacklistIface {
		blacklist = append(blacklist, b.(string))
	}

	connectedNodes, err := utils.GetConnectedNodesAddresses(saiP2Paddress, blacklist)
	if err != nil {
		s.GlobalService.Logger.Error(err.Error())
		return err
	}
	// 3 ways to form sync request
	syncRequest, err := s.formSyncRequest(blockNumber, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error(err.Error())
		return err
	}
	syncRequest.Address = s.IpAddress

	//todo : func is not ready yet
	for _, node := range connectedNodes {
		err := utils.SendDirectGetBlockMsg(node, syncRequest, saiP2Paddress)
		if err != nil {
			s.GlobalService.Logger.Error(err.Error())
			return err
		}
		//missedBlocks <-

	}
	return nil

}

// detect and form syncRequest to get missed blocks
func (s *InternalService) formSyncRequest(blockNumber int, storageToken string) (*models.SyncRequest, error) {
	opts := options.Find().SetSort(bson.M{"block.number": -1}).SetLimit(1)
	err, result := s.Storage.Get(blockchainCol, bson.M{}, opts, storageToken)
	if err != nil {
		s.GlobalService.Logger.Error("chain - handleBlockCandidate - formSyncRequest - get last block", zap.Error(err))
		return nil, err
	}

	//means that we havent blocks in our blockchain collection
	if len(result) == 2 {
		return &models.SyncRequest{
			From:    0,
			To:      blockNumber,
			Address: s.IpAddress,
		}, nil
	}

	blocks := make([]*models.BlockConsensusMessage, 0)
	data, err := utils.ExtractResult(result)
	if err != nil {
		Service.GlobalService.Logger.Error("chain - handleBlockCandidate - formSyncRequest - extract data from response", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal(data, &blocks)
	if err != nil {
		s.GlobalService.Logger.Error("chain - handleBlockCandidate - formSyncRequest - unmarshal result of last block from blockchain collection", zap.Error(err))
		return nil, err
	}
	block := blocks[0]
	currentBlock := block.Block.Number

	// we need only 1 exact block
	if currentBlock == blockNumber {
		return &models.SyncRequest{
			From:    blockNumber,
			To:      blockNumber,
			Address: s.IpAddress,
		}, nil
	}

	// node is lagging behind, requesting missing blocks
	if currentBlock < blockNumber {
		return &models.SyncRequest{
			From:    currentBlock,
			To:      blockNumber,
			Address: s.IpAddress,
		}, nil
	} else {
		return &models.SyncRequest{
			From:    block.Block.Number,
			To:      blockNumber,
			Address: s.IpAddress,
		}, nil
	}

}

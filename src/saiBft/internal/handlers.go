// implementing handlers
package internal

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/bft/utils"
	"github.com/iamthe1whoknocks/saiService"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// connect saiP2p node to our microservice
var ConnectSaiP2pNodeHandler = saiService.HandlerElement{
	Name:        "connect",
	Description: "connect saiP2p node",
	Function: func(data interface{}) (interface{}, error) {
		address, ok := data.(string)
		if !ok {
			Service.Logger.Debug("handling connect method, wrong type")
			return "not ok", nil
		}
		Service.Logger.Debug("got from connect handler", zap.String("value", string(address)))

		Service.Mutex.Lock()
		Service.ConnectedSaiP2pNodes[address] = &models.SaiP2pNode{
			Address: address,
		}
		Service.Mutex.Unlock()
		Service.Logger.Sugar().Debugf("new saiP2p node was connected, connected nodes : %v", Service.ConnectedSaiP2pNodes)
		return "ok", nil
	},
}

// connect saiP2p node to our microservice
var GetMissedBlocks = saiService.HandlerElement{
	Name:        "getBlocks",
	Description: "get missed blocks",
	Function: func(data interface{}) (interface{}, error) {
		blockNumber, ok := data.(float64)
		if !ok {
			err := fmt.Errorf("wrong type of incoming data,incoming data : %s", data)
			Service.Logger.Error("handlers - GetMissedBlocks - type assertion to GetBlocksRequest", zap.Error(err))
			return nil, fmt.Errorf("wrong type of incoming data")
		}

		storageToken, ok := Service.GlobalService.Configuration["storage_token"].(string)
		if !ok {
			Service.Logger.Fatal("wrong type of storage_token value in config")
		}
		filterGte := bson.M{"block.number": bson.M{"$lte": blockNumber}}
		err, response := DB.storage.Get(blockchainCollection, filterGte, bson.M{}, storageToken)
		if err != nil {
			Service.Logger.Error("handlers - GetMissedBlocks - get blocks from storage", zap.Error(err))
			return nil, fmt.Errorf("handlers - GetMissedBlocks - get blocks from storage : %w", err)
		}

		if len(response) == 2 {
			err = fmt.Errorf("block with number = %f was not found", blockNumber)
			Service.Logger.Error("handleBlockConsensusMsg - get block N", zap.Error(err))
			return nil, err
		}

		result, err := utils.ExtractResult(response)
		if err != nil {
			Service.Logger.Error("handlers - GetMissedBlocks - get blocks from storage - extract result", zap.Error(err))
			return nil, fmt.Errorf("handlers - GetMissedBlocks - get blocks from storage - extract result: %w", err)
		}
		blocks := make([]*models.BlockConsensusMessage, 0)

		err = json.Unmarshal(result, &blocks)
		if err != nil {
			Service.Logger.Error("handlers - GetMissedBlocks - get blocks from storage - unmarshal result", zap.Error(err))
			return nil, fmt.Errorf("handlers - GetMissedBlocks - get blocks from storage - unmarshal result: %w", err)
		}
		return blocks, nil
	},
}

// connect saiP2p node to our microservice
var HandleMessage = saiService.HandlerElement{
	Name:        "message",
	Description: "handle messages",
	Function: func(data interface{}) (interface{}, error) {
		Service.Logger.Debug("got message from cli", zap.String("data", data.(string)))
		Service.MsgQueue <- data
		time.Sleep(10 * time.Second) // for test purposes
		return "ok", nil
	},
}

func (s *InternalService) Init() {
	go s.listenFromSaiP2P(s.GlobalService.Configuration["saiBTC_address"].(string))

}

func (s *InternalService) Process() {
	s.Processing()
}

// implementing handlers
package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/bft/utils"
	"github.com/iamthe1whoknocks/saiService"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// get missed blocks
var GetMissedBlocks = saiService.HandlerElement{
	Name:        "getBlocks",
	Description: "get missed blocks",
	Function: func(data interface{}) (interface{}, error) {
		cliData, ok := data.([]string)
		if !ok {
			err := fmt.Errorf("wrong type of incoming data,incoming data : %s, type : %+v", data, reflect.TypeOf(data))
			Service.GlobalService.Logger.Error("handlers - GetMissedBlocks - type assertion to GetBlocksRequest", zap.Error(err))
			return nil, fmt.Errorf("wrong type of incoming data")
		}

		storageToken, ok := Service.GlobalService.Configuration["storage_token"].(string)
		if !ok {
			Service.GlobalService.Logger.Fatal("wrong type of storage_token value in config")
		}

		if len(cliData) == 0 {
			err := errors.New("empty argument provided")
			Service.GlobalService.Logger.Error("handlers - getBlocks", zap.Error(err))
			return nil, err
		}

		num := cliData[0]
		blockNumber, err := strconv.Atoi(num)
		if err != nil {
			Service.GlobalService.Logger.Sugar().Fatalf("Cant convert cli input to int  :%s", err.Error())
		}

		filterGte := bson.M{"block.number": bson.M{"$lte": blockNumber}}
		err, response := Service.Storage.Get(blockchainCollection, filterGte, bson.M{}, storageToken)
		if err != nil {
			Service.GlobalService.Logger.Error("handlers - GetMissedBlocks - get blocks from storage", zap.Error(err))
			return nil, fmt.Errorf("handlers - GetMissedBlocks - get blocks from storage : %w", err)
		}

		if len(response) == 2 {
			err = fmt.Errorf("block with number = %d was not found", blockNumber)
			Service.GlobalService.Logger.Error("handleBlockConsensusMsg - get block N", zap.Error(err))
			return nil, err
		}

		result, err := utils.ExtractResult(response)
		if err != nil {
			Service.GlobalService.Logger.Error("handlers - GetMissedBlocks - get blocks from storage - extract result", zap.Error(err))
			return nil, fmt.Errorf("handlers - GetMissedBlocks - get blocks from storage - extract result: %w", err)
		}
		blocks := make([]*models.BlockConsensusMessage, 0)

		err = json.Unmarshal(result, &blocks)
		if err != nil {
			Service.GlobalService.Logger.Error("handlers - GetMissedBlocks - get blocks from storage - unmarshal result", zap.Error(err))
			return nil, fmt.Errorf("handlers - GetMissedBlocks - get blocks from storage - unmarshal result: %w", err)
		}
		return blocks, nil
	},
}

// handle tx message from cli
// example : bft tx send $FROM $TO $AMOUNT $DENOM
var HandleTxFromCli = saiService.HandlerElement{
	Name:        "tx",
	Description: "handle tx message",
	Function: func(data interface{}) (interface{}, error) {
		args, ok := data.([]string)
		if !ok {
			return nil, errors.New("wrong type for args in cli tx method")
		}
		Service.GlobalService.Logger.Debug("got message from cli", zap.Strings("data", args))

		if len(args) != 5 {
			return nil, errors.New("not enough arguments in cli tx method")
		}

		saiBtcAddress, ok := Service.GlobalService.Configuration["saiBTC_address"].(string)
		if !ok {
			Service.GlobalService.Logger.Fatal("wrong type of saiBTC_address value in config")
		}

		btckeys, err := Service.GetBTCkeys("btc_keys.json", saiBtcAddress)
		if err != nil {
			Service.GlobalService.Logger.Fatal("listenFromSaiP2P  - handle tx msg - get btc keys", zap.Error(err))
		}
		Service.BTCkeys = btckeys

		txMsg := &models.TxMessage{
			Method: args[0],
		}
		params := args[1:]
		txMsg.Params = append(txMsg.Params, params...)

		//todo : tx msg in bytes or struct, not string
		txMsgBytes, err := json.Marshal(txMsg)
		if err != nil {
			Service.GlobalService.Logger.Error("handlers - tx  -  marshal tx msg", zap.Error(err))
			return nil, fmt.Errorf("handlers - tx  -  marshal tx msg: %w", err)
		}
		transactionMessage := &models.TransactionMessage{
			Tx: &models.Tx{
				SenderAddress: Service.BTCkeys.Address,
				Message:       string(txMsgBytes), //todo : tx msg in bytes or struct, not string
			},
		}

		hash, err := transactionMessage.Tx.GetHash()
		if err != nil {
			Service.GlobalService.Logger.Error("handlers  - tx - count tx message hash", zap.Error(err))
			return nil, fmt.Errorf("handlers  - tx - count tx message hash: %w", err)
		}
		transactionMessage.Tx.MessageHash = hash

		btcResp, err := utils.SignMessage(transactionMessage, saiBtcAddress, Service.BTCkeys.Private)
		if err != nil {
			Service.GlobalService.Logger.Error("handlers  - tx - sign tx message", zap.Error(err))
			return nil, fmt.Errorf("handlers  - tx - sign tx message: %w", err)
		}
		transactionMessage.Tx.SenderSignature = btcResp.Signature

		Service.MsgQueue <- transactionMessage.Tx
		<-Service.MsgQueue
		return "ok", nil
	},
}

// handle message from saiP2p
var HandleMessage = saiService.HandlerElement{
	Name:        "message",
	Description: "handle  message from saiP2p",
	Function: func(data interface{}) (interface{}, error) {
		m, ok := data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Wrong type of data  : %+v\n", reflect.TypeOf(data))
		}
		Service.GlobalService.Logger.Sugar().Debugf("got message from saiP2p : %+v", m) // DEBUG

		switch m["type"].(string) {
		case models.BlockConsensusMsgType:
			msg := models.BlockConsensusMessage{}
			b, err := json.Marshal(m)
			if err != nil {
				return nil, fmt.Errorf("handlers - handle message - unmarshal : %w", err)
			}
			err = json.Unmarshal(b, &msg)
			if err != nil {
				return nil, fmt.Errorf("handlers - handle message - marshal bytes : %w", err)
			}
			Service.MsgQueue <- &msg
		case models.ConsensusMsgType:
			msg := models.ConsensusMessage{}
			b, err := json.Marshal(m)
			if err != nil {
				return nil, fmt.Errorf("handlers - handle message - unmarshal : %w", err)
			}
			err = json.Unmarshal(b, &msg)
			if err != nil {
				return nil, fmt.Errorf("handlers - handle message - marshal bytes : %w", err)
			}
			Service.MsgQueue <- &msg
		case models.TransactionMsgType:
			msg := models.Tx{}
			b, err := json.Marshal(m)
			if err != nil {
				return nil, fmt.Errorf("handlers - handle message - unmarshal : %w", err)
			}
			err = json.Unmarshal(b, &msg)
			if err != nil {
				return nil, fmt.Errorf("handlers - handle message - marshal bytes : %w", err)
			}
			Service.MsgQueue <- &msg
		default:
			return nil, errors.New("handlers - handle message - wrong message type" + m["type"].(string))
		}

		return "ok", nil
	},
}

func (s *InternalService) Init() {
	go s.listenFromSaiP2P(s.GlobalService.Configuration["saiBTC_address"].(string))

}

func (s *InternalService) Process() {
	s.Processing()
}

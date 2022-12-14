// implementing handlers
package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/bft/utils"
	"github.com/iamthe1whoknocks/saiService"
	"go.uber.org/zap"
)

// get missed blocks
var GetMissedBlocks = saiService.HandlerElement{
	Name:        "GetMissedBlocksResponse",
	Description: "get missed blocks from another node",
	Function: func(data interface{}) (interface{}, error) {
		respData, ok := data.(map[string]interface{})
		if !ok {
			err := fmt.Errorf("wrong type of incoming data,incoming data : %s, type : %+v", data, reflect.TypeOf(data))
			Service.GlobalService.Logger.Error("handlers - GetMissedBlocks - type assertion to GetBlocksRequest", zap.Error(err))
			return nil, fmt.Errorf("wrong type of incoming data")
		}

		Service.GlobalService.Logger.Debug("handlers - GetMissedBlocksResponse - get sync response", zap.Any("raw response data", respData))

		dataInBytes, err := json.Marshal(respData)
		if err != nil {
			Service.GlobalService.Logger.Debug("handlers - GetMissedBlocksResponse - marshal sync response", zap.Error(err))
			return nil, err
		}
		syncResponse := &models.SyncResponse{}

		err = json.Unmarshal(dataInBytes, &syncResponse)
		if err != nil {
			Service.GlobalService.Logger.Error("handlers - GetMissedBlocksResponse - unmarshal response", zap.Error(err))
			return nil, err
		}

		if syncResponse.Error != nil {
			Service.GlobalService.Logger.Error("handlers - GetMissedBlocksResponse - error from syncResponse", zap.Error(syncResponse.Error))
			return nil, syncResponse.Error
		}

		if syncResponse.Type != models.SyncResponseType {
			Service.GlobalService.Logger.Error("handlers - GetMissedBlocksResponse - wrong type for msg", zap.String("incoming type", syncResponse.Type))
			return nil, syncResponse.Error
		}

		Service.MissedBlocksLinkCh <- syncResponse.Link
		return "ok", nil
	},
}

// handle tx message from cli
// example : bft tx send $FROM $TO $AMOUNT $DENOM
var HandleTxFromCli = saiService.HandlerElement{
	Name:        "tx",
	Description: "handle tx message",
	Function: func(data interface{}) (interface{}, error) {
		argsStr := make([]string, 0)
		txStruct := models.TxFromHandler{
			IsFromCli: false,
		}
		switch args := data.(type) {
		case []string:
			argsStr = args
			Service.GlobalService.Logger.Debug("got message from cli", zap.Strings("data", argsStr))
			txStruct.IsFromCli = true
		case []interface{}:
			for _, iface := range args {
				strArg, ok := iface.(string)
				if !ok {
					return nil, fmt.Errorf("wrong argument type in tx data, argument : [%s],type  :[%s]", iface, reflect.TypeOf(data))
				}
				argsStr = append(argsStr, strArg)
			}
			Service.GlobalService.Logger.Debug("got message from http", zap.Strings("data", argsStr))
		default:
			return nil, fmt.Errorf("wrong type for args in cli tx method, current type :%s", reflect.TypeOf(data))
		}

		Service.GlobalService.Logger.Debug("got message from cli", zap.Strings("data", argsStr))

		if len(argsStr) != 5 {
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
			Method: argsStr[0],
		}
		params := argsStr[1:]
		txMsg.Params = append(txMsg.Params, params...)

		txMsgBytes, err := json.Marshal(txMsg)
		if err != nil {
			Service.GlobalService.Logger.Error("handlers - tx  -  marshal tx msg", zap.Error(err))
			return nil, fmt.Errorf("handlers - tx  -  marshal tx msg: %w", err)
		}
		transactionMessage := &models.TransactionMessage{

			Tx: &models.Tx{
				Type:          models.TransactionMsgType,
				SenderAddress: Service.BTCkeys.Address,
				Message:       string(txMsgBytes),
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

		saiP2Paddress, ok := Service.GlobalService.Configuration["saiP2P_address"].(string)
		if !ok {
			Service.GlobalService.Logger.Fatal("processing - wrong type of saiP2P address value from config")
		}

		err = Service.broadcastMsg(transactionMessage.Tx, saiP2Paddress)
		if err != nil {
			Service.GlobalService.Logger.Error("listenFromSaiP2P  - handle tx msg - broadcast tx", zap.Error(err))
		}

		txStruct.Tx = transactionMessage.Tx

		Service.MsgQueue <- &txStruct

		if txStruct.IsFromCli {
			<-Service.TxHandlerSyncCh
		}

		return "ok", nil
	},
}

// handle message from saiP2p
var HandleMessage = saiService.HandlerElement{
	Name:        "message",
	Description: "handle message from saiP2p",
	Function: func(data interface{}) (interface{}, error) {
		m, ok := data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Wrong type of data  : %+v\n", reflect.TypeOf(data))
		}
		Service.GlobalService.Logger.Sugar().Debugf("got message from saiP2p : %+v", m) // DEBUG

		switch m["type"].(string) {
		case models.BlockConsensusMsgType:
			Service.GlobalService.Logger.Sugar().Debugf("got message from saiP2p detected type : %s", models.BlockConsensusMsgType) // DEBUG
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
			Service.GlobalService.Logger.Sugar().Debugf("got message from saiP2p detected type : %s", models.ConsensusMsgType) // DEBUG
			msg := models.ConsensusMessage{}
			b, err := json.Marshal(m)
			if err != nil {
				Service.GlobalService.Logger.Sugar().Error(err) // DEBUG
				return nil, fmt.Errorf("handlers - handle message - unmarshal : %w", err)
			}
			err = json.Unmarshal(b, &msg)
			if err != nil {
				Service.GlobalService.Logger.Sugar().Error(err) // DEBUG
				return nil, fmt.Errorf("handlers - handle message - marshal bytes : %w", err)
			}
			Service.MsgQueue <- &msg
		case models.TransactionMsgType:
			Service.GlobalService.Logger.Sugar().Debugf("got message from saiP2p detected type : %s", models.TransactionMsgType) // DEBUG
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
			Service.GlobalService.Logger.Sugar().Errorf("got message from saiP2p wrong detected type : %s", m["type"].(string)) // DEBUG
			return nil, errors.New("handlers - handle message - wrong message type" + m["type"].(string))
		}

		return "ok", nil
	},
}

// create btc keys
// example : keys
var CreateBTCKeys = saiService.HandlerElement{
	Name:        "keys",
	Description: "create btc keys",
	Function: func(data interface{}) (interface{}, error) {
		args, ok := data.([]string)
		if !ok {
			return nil, errors.New("wrong type for args in cli keys method")
		}
		Service.GlobalService.Logger.Debug("got message from cli", zap.Strings("data", args))

		// todo : handle flags if it will be needed
		// if len(args) != 5 {
		// 	return nil, errors.New("not enough arguments in cli tx method")
		// }
		file, err := os.OpenFile(btcKeyFile, os.O_RDWR, 0666)

		//todo: handle args if file exists
		if err == nil {
			Service.GlobalService.Logger.Debug("handlers - create btc keys - open key btc file - keys already exists")
			return "btc keys file already exists", nil
		}

		saiBtcAddress, ok := Service.GlobalService.Configuration["saiBTC_address"].(string)
		if !ok {
			Service.GlobalService.Logger.Fatal("wrong type of saiBTC_address value in config")
		}

		btcKeys, body, err := utils.GetBtcKeys(saiBtcAddress)
		if err != nil {
			Service.GlobalService.Logger.Error("handlers - create btc keys  - get btc keys", zap.Error(err))
			return nil, err
		}

		file, err = os.OpenFile(btcKeyFile, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			Service.GlobalService.Logger.Error("handlers - create btc keys  - open key btc file", zap.Error(err))
			return nil, err
		}
		_, err = file.Write(body)
		if err != nil {
			Service.GlobalService.Logger.Error("handlers -  - write btc keys to file", zap.Error(err))
			return nil, err
		}
		return btcKeys, nil

	},
}

func (s *InternalService) Init() {
	go s.listenFromSaiP2P(s.GlobalService.Configuration["saiBTC_address"].(string))

}

func (s *InternalService) Process() {
	s.Processing()
}

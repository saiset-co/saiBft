package internal

import (
	"sync"

	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/bft/utils"
	"github.com/iamthe1whoknocks/saiService"
	"go.uber.org/zap"
)

// here we add all implemented handlers, create name of service and register config
// moved from handlers to service because of initialization problems
func Init(svc *saiService.Service) {
	storage := NewDB()
	Service.Storage = storage

	btckeys, err := Service.GetBTCkeys("btc_keys.json", Service.GlobalService.Configuration["saiBTC_address"].(string))
	if err != nil {
		svc.Logger.Fatal("main - init - open btc keys", zap.Error(err))
	}
	Service.BTCkeys = btckeys

	svc.Logger.Sugar().Debugf("btc keys : %+v\n", Service.BTCkeys) //DEBUG

	Service.Handler[GetMissedBlocks.Name] = GetMissedBlocks
	Service.Handler[HandleTxFromCli.Name] = HandleTxFromCli
	Service.Handler[HandleMessage.Name] = HandleMessage
}

type InternalService struct {
	Handler              saiService.Handler  // handlers to define in this specified microservice
	GlobalService        *saiService.Service // saiService reference
	TrustedValidators    []string
	Mutex                *sync.RWMutex
	ConnectedSaiP2pNodes map[string]*models.SaiP2pNode
	BTCkeys              *models.BtcKeys
	MsgQueue             chan interface{}
	Storage              utils.Database
}

// global handler for registering handlers
var Service = &InternalService{
	Handler:              saiService.Handler{},
	Mutex:                new(sync.RWMutex),
	ConnectedSaiP2pNodes: make(map[string]*models.SaiP2pNode),
	MsgQueue:             make(chan interface{}),
}

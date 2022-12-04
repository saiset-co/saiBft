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

	Service.IpAddress = utils.GetOutboundIP()
	if Service.IpAddress == "" {
		svc.Logger.Fatal("Cannot detect outbound IP address of node")
	}

	svc.Logger.Debug("node address : ", zap.String("ip address", Service.IpAddress)) //DEBUG
	svc.Logger.Sugar().Debugf("btc keys : %+v\n", Service.BTCkeys)                   //DEBUG

	skipInitializating, ok := Service.GlobalService.Configuration["skip_initializating"].(bool)
	if !ok {
		svc.Logger.Fatal("handlers - processing - wrong type of skip initializating value from config")
	}

	Service.SkipInitializating = skipInitializating

	Service.Handler[GetMissedBlocks.Name] = GetMissedBlocks
	Service.Handler[HandleTxFromCli.Name] = HandleTxFromCli
	Service.Handler[HandleMessage.Name] = HandleMessage
	Service.Handler[CreateBTCKeys.Name] = CreateBTCKeys
}

type InternalService struct {
	Handler              saiService.Handler  // handlers to define in this specified microservice
	GlobalService        *saiService.Service // saiService reference
	TrustedValidators    []string
	Mutex                *sync.RWMutex
	ConnectedSaiP2pNodes map[string]*models.SaiP2pNode
	BTCkeys              *models.BtcKeys
	MsgQueue             chan interface{}
	InitialSignalCh      chan struct{} // chan for notification, if initial block consensus msg was got already
	IsInitialized        bool          // if inital block consensus msg was got or timeout was passed
	Storage              utils.Database
	IpAddress            string // outbound ip address
	MissedBlocksQueue    chan *models.SyncResponse
	SkipInitializating   bool // first node mode, if true
	Round7State          bool // if process at the moment at 7 round
	GoToStartLoopCh      chan struct{}
}

// global handler for registering handlers
var Service = &InternalService{
	Handler:              saiService.Handler{},
	Mutex:                new(sync.RWMutex),
	ConnectedSaiP2pNodes: make(map[string]*models.SaiP2pNode),
	MsgQueue:             make(chan interface{}),
	MissedBlocksQueue:    make(chan *models.SyncResponse),
	InitialSignalCh:      make(chan struct{}),
	IsInitialized:        false,
	SkipInitializating:   false,
	Round7State:          false,
	GoToStartLoopCh:      make(chan struct{}),
}

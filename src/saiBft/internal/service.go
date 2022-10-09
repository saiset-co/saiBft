package internal

import (
	"sync"

	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/saiService"
	"go.uber.org/zap"
)

// here we add all implemented handlers, create name of service and register config
// moved from handlers to service because of initialization problems
func Init(svc *saiService.Service, logger *zap.Logger) {
	MicroserviceConfiguration = NewConfiguration()
	MicroserviceConfiguration.Set(svc.Configuration)
	DB = NewDB()
	Service.Logger = logger

	Service.Handler[ConnectSaiP2pNodeHandler.Name] = ConnectSaiP2pNodeHandler
}

type InternalService struct {
	Handler              saiService.Handler  // handlers to define in this specified microservice
	GlobalService        *saiService.Service // saiService reference
	Logger               *zap.Logger
	TrustedValidators    []string
	Mutex                *sync.RWMutex
	ConnectedSaiP2pNodes map[string]*models.SaiP2pNode
	BTCkeys              *models.BtcKeys
}

// global handler for registering handlers
var Service = &InternalService{
	Handler:              saiService.Handler{},
	Mutex:                new(sync.RWMutex),
	ConnectedSaiP2pNodes: make(map[string]*models.SaiP2pNode),
	Logger:               &zap.Logger{},
}

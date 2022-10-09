// implementing handlers
package internal

import (
	"fmt"

	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/saiService"
	"go.uber.org/zap"
)

// connect saiP2p node to our microservice
var ConnectSaiP2pNodeHandler = saiService.HandlerElement{
	Name:        "ConnectSaiP2pNode",
	Description: "connect saiP2p node",
	Function: func(data interface{}) (interface{}, error) {
		saiP2pNode, ok := data.(*models.SaiP2pNode)
		if !ok {
			err := fmt.Errorf("wrong type of incoming data,incoming data : %s", data)
			Service.Logger.Error("handlers - connectSaiP2pNode - type assertion to saiP2pNode", zap.Error(err))
			return nil, fmt.Errorf("wrong type of incoming data")
		}
		Service.Mutex.Lock()
		Service.ConnectedSaiP2pNodes[saiP2pNode.Address] = saiP2pNode
		Service.Mutex.Unlock()
		Service.Logger.Sugar().Debugf("new saiP2p node was connected, connected nodes : %v", Service.ConnectedSaiP2pNodes)
		return nil, nil
	},
}

func (s *InternalService) Init() {
}

func (s *InternalService) Process() {
	s.Processing()
}

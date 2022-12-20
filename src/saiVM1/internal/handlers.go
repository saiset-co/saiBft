package internal

import (
	"github.com/saiset-co/saiService"
	"go.mongodb.org/mongo-driver/bson"
)

func (is InternalService) NewHandler() saiService.Handler {
	return saiService.Handler{
		"execute": saiService.HandlerElement{
			Name:        "execute",
			Description: "Execute smart-contract",
			Function: func(data interface{}) (interface{}, error) {
				return is.execute(data), nil
			},
		},
	}
}

func (is InternalService) execute(data interface{}) interface{} {
	counter++
	return bson.M{"vm_processed": true, "vm_result": true, "vm_response": bson.M{"callNumber": counter, "C": bson.A{"123", "Matok", "777"}}}
}

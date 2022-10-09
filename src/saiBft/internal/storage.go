package internal

import (
	"log"

	"github.com/iamthe1whoknocks/bft/utils"
)

// example storage
var DB *storageManager

// Example of storage to deal with in handlers.
// Must be global, if handler func will be like func() interface{}
type storageManager struct {
	storage utils.Database
}

func NewDB() *storageManager {
	url, ok := MicroserviceConfiguration.Cfg["storage_url"].(string)
	if !ok {
		log.Fatalf("configuration : invalid storage url provided, url : %s", MicroserviceConfiguration.Cfg["storage_url"])
	}
	email, ok := MicroserviceConfiguration.Cfg["storage_email"].(string)
	if !ok {
		log.Fatalf("configuration : invalid storage email provided, email : %s", MicroserviceConfiguration.Cfg["storage_email"])
	}
	password, ok := MicroserviceConfiguration.Cfg["storage_password"].(string)
	if !ok {
		log.Fatalf("configuration : invalid storage password provided, password : %s", MicroserviceConfiguration.Cfg["storage_email"])
	}

	s := &storageManager{
		storage: utils.Storage(url, email, password),
	}
	return s
}

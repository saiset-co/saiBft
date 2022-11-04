package internal

import (
	"log"

	"github.com/iamthe1whoknocks/bft/utils"
)

func NewDB() utils.Database {
	url, ok := Service.GlobalService.Configuration["storage_url"].(string)
	if !ok {
		log.Fatalf("configuration : invalid storage url provided, url : %s", Service.GlobalService.Configuration["storage_url"])
	}
	email, ok := Service.GlobalService.Configuration["storage_email"].(string)
	if !ok {
		log.Fatalf("configuration : invalid storage email provided, email : %s", Service.GlobalService.Configuration["storage_email"])
	}
	password, ok := Service.GlobalService.Configuration["storage_password"].(string)
	if !ok {
		log.Fatalf("configuration : invalid storage password provided, password : %s", Service.GlobalService.Configuration["storage_email"])
	}

	return utils.Storage(url, email, password)
}

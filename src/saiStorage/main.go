package main

import (
	"github.com/iamthe1whoknocks/saiStorage/config"
	"github.com/iamthe1whoknocks/saiStorage/mongo"
	"github.com/iamthe1whoknocks/saiStorage/server"
)

func main() {
	cfg := config.Load()
	srv := server.NewServer(cfg, false)
	mSrv := mongo.NewMongoServer(cfg)

	go mSrv.Start()

	srv.Start()
	//srv.StartHttps()
}

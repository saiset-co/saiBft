package main

import (
	"fmt"
	"log"
	"saiP2p/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config, err := NewConfig("config.yml")
	if err != nil {
		log.Fatalf("Open config file : %s", err)
	}

	storage := utils.Storage(config.StorageURL, config.StorageEmail, config.StoragePassword)

	proxy := &Proxy{
		Config:  config,
		Storage: &storage,
	}
	proxy.SetLogger()

	err = proxy.createFileDirectory()
	if err != nil {
		log.Fatalf("create file directory : %s", err)
	}

	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(cors.Default())
	cfg := cors.DefaultConfig()
	cfg.AllowAllOrigins = true

	r.POST("/send", proxy.Handler)
	r.GET("/check", proxy.check)
	r.GET("/sync", proxy.sync)
	r.Static("/files", "./files")

	r.Run(fmt.Sprintf("%s:%s", config.Host, config.Port))
}

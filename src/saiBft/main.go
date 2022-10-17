package main

import (
	"log"

	"github.com/iamthe1whoknocks/bft/internal"
	"github.com/iamthe1whoknocks/saiService"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment(zap.AddStacktrace(zap.DPanicLevel))

	if err != nil {
		log.Fatalf("error when initialize logger : %w", err)
	}
	svc := saiService.NewService("bft")

	svc.RegisterConfig("build/config.yml")
	internal.Init(svc, logger)
	svc.Logger = logger

	internal.Service.GlobalService = svc

	internal.Service.GlobalService.RegisterHandlers(internal.Service.Handler)

	internal.Service.GlobalService.RegisterInitTask(internal.Service.Init)

	internal.Service.GlobalService.RegisterTasks([]func(){
		internal.Service.Process,
	})
	internal.Service.GlobalService.Start()

}

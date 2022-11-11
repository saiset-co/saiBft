package main

import (
	"github.com/iamthe1whoknocks/bft/internal"
	"github.com/iamthe1whoknocks/saiService"
)

func main() {
	svc := saiService.NewService("bft")

	internal.Service.GlobalService = svc

	svc.RegisterConfig("config.yml")

	internal.Init(svc)

	internal.Service.GlobalService.RegisterHandlers(internal.Service.Handler)

	internal.Service.GlobalService.RegisterInitTask(internal.Service.Init)

	internal.Service.GlobalService.RegisterTasks([]func(){
		internal.Service.Process,
	})

	internal.Service.GlobalService.Start()

}

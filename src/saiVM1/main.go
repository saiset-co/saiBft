package main

import (
	"github.com/saiset-co/Boilerplate/internal"
	"github.com/saiset-co/saiService"
)

func main() {
	svc := saiService.NewService("sai_VM1")
	is := internal.InternalService{Context: svc.Context}

	svc.RegisterConfig("config.yml")

	svc.RegisterHandlers(
		is.NewHandler(),
	)

	svc.Start()

}

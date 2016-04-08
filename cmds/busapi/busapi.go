package main

import (
	"log"
	"net/http"

	"github.com/kelseyhightower/envconfig"

	"github.com/brnstz/bus/api"
	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
)

func main() {
	var err error
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	err = envconfig.Process("bus", &conf.DB)
	if err != nil {
		log.Fatal(err)
	}

	err = envconfig.Process("bus", &conf.API)
	if err != nil {
		log.Fatal(err)
	}

	etc.DBConn = etc.MustDB()

	handler := api.NewHandler()

	log.Fatal(http.ListenAndServe(conf.API.Addr, handler))
}

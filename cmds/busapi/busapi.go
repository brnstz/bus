package main

import (
	"log"
	"net/http"
	"time"

	_ "net/http/pprof"

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

	time.Local, err = time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}

	etc.DBConn = etc.MustDB()

	if conf.API.BuildTimestamp == 0 {
		conf.API.BuildTimestamp = time.Now().Unix()
	}

	handler := api.NewHandler()

	err = api.InitRouteCache()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	log.Fatal(http.ListenAndServe(conf.API.Addr, handler))
}

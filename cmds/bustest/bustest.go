package main

import (
	"log"
	"net/http"
	"time"

	_ "net/http/pprof"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/kelseyhightower/envconfig"
)

func main() {
	var err error
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	err = envconfig.Process("bus", &conf.DB)
	if err != nil {
		log.Fatal(err)
	}

	err = envconfig.Process("bus", &conf.Loader)
	if err != nil {
		log.Fatal(err)
	}

	time.Local, err = time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}

	etc.DBConn = etc.MustDB()

	refreshViews := []string{
		//		"REFRESH MATERIALIZED VIEW here;",
		"REFRESH MATERIALIZED VIEW service;",
		"REFRESH MATERIALIZED VIEW service_exception;",
	}

	for _, refreshView := range refreshViews {
		_, err = etc.DBConn.Exec(refreshView)
		if err != nil {
			log.Fatal(err)
		}
	}

}

package main

import (
	"log"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/loader"
	"github.com/kelseyhightower/envconfig"
)

func main() {
	var err error
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	err = envconfig.Process("bus", &conf.DB)
	if err != nil {
		log.Fatal(err)
	}

	err = envconfig.Process("bus", &conf.Loader)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(conf.Loader)

	etc.DBConn = etc.MustDB()

	if conf.Loader.LoadForever {
		loader.LoadForever(conf.Loader.RouteFilter, conf.Loader.GTFSURLs...)
	} else {
		loader.LoadOnce(conf.Loader.RouteFilter, conf.Loader.GTFSURLs...)
	}
}

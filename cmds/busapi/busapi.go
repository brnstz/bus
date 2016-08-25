package main

import (
	"log"
	"net/http"
	"time"

	"github.com/NYTimes/gziphandler"
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

	err = envconfig.Process("bus", &conf.Cache)
	if err != nil {
		log.Fatal(err)
	}

	err = envconfig.Process("bus", &conf.Partner)
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

	withgz := gziphandler.GzipHandler(handler)

	err = api.InitRouteCache()
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(http.ListenAndServe(conf.API.Addr, withgz))
}

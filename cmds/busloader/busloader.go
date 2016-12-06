package main

import (
	"log"
	"os"
	"time"

	"github.com/brnstz/upsert"
	"github.com/kelseyhightower/envconfig"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/loader"
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

	time.Local, err = time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}

	etc.DBConn = etc.MustDB()

	upsert.LongQuery = time.Duration(1 * time.Second)
	//upsert.Debug = true

	// If we specified a specific temp dir, clean it up first. This prevents a
	// series of crashes from filling up the disk.
	tmpdir := os.Getenv("TMPDIR")
	if len(tmpdir) > 0 {
		os.RemoveAll(tmpdir)
		os.MkdirAll(tmpdir, 0775)
	}

	if conf.Loader.LoadForever {
		loader.LoadForever()
	} else {
		loader.LoadOnce()
	}
}

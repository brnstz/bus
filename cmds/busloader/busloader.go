package main

import (
	"log"
	"os"
	"strings"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/loader"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	conf.ConfigVar(&conf.DBAddr, "localhost:5432", "BUS_DB_ADDR", true)
	conf.ConfigVar(&conf.DBUser, "postgres", "BUS_DB_USER", true)
	conf.ConfigVar(&conf.DBName, "postgres", "BUS_DB_Name", true)
	conf.ConfigVar(&conf.TmpDir, os.TempDir(), "BUS_TMP_DIR", true)
	conf.ConfigVar(&conf.GTFSURLs, "", "BUS_GTFS_URLS", true)
	conf.ConfigVar(&conf.LoadForever, "true", "BUS_LOAD_FOREVER", true)
	conf.ConfigVar(&conf.RouteFilter, "", "BUS_ROUTE_FILTER", false)

	conf.DB = conf.MustDB()

	urls := strings.Split(conf.GTFSURLs, "|")

	switch conf.LoadForever {
	case "true":
		loader.LoadForever(conf.RouteFilter, urls...)
	case "false":
		log.Println(conf.RouteFilter, urls)
		loader.LoadOnce(conf.RouteFilter, urls...)

	default:
		log.Fatalf("invalid value for $BUS_LOAD_FOREVER: %v", conf.LoadForever)
	}
}

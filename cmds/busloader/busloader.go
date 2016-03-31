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
	conf.ConfigVar(&conf.TmpDir, os.TempDir(), "BUS_TMP_DIR", true)
	conf.ConfigVar(&conf.GTFSURLs, "", "BUS_GTFS_URLS", true)
	conf.ConfigVar(&conf.RouteFilter, "", "BUS_ROUTE_FILTER", false)

	conf.DB = conf.MustDB()

	urls := strings.Split(conf.GTFSURLs, "|")

	loader.LoadForever(conf.RouteFilter, urls...)
}

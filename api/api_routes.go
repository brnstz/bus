package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
)

// FIXME: this is a temporary hack that will only work with one city

var (
	routeCache []byte
)

type routesResp struct {
	Routes []*models.Route `json:"routes"`
}

func InitRouteCache() error {
	// getAll subway/train routes so we can pre-render them. Including
	// buses would be too much
	routes, err := models.GetPreloadRoutes(etc.DBConn, conf.Partner.AgencyIDs)
	if err != nil {
		log.Println("can't get routes", err)
		return err
	}

	rr := routesResp{
		Routes: routes,
	}

	b, err := json.Marshal(rr)
	if err != nil {
		log.Println("can't marshal routes", err)
		return err
	}

	routeCache = b
	return nil
}

func getRoutes(w http.ResponseWriter, r *http.Request) {
	var err error

	if len(routeCache) < 0 {
		log.Println("routeCache should be initialized before API is started")
		err = InitRouteCache()
		if err != nil {
			log.Println("couldn't init route cache", err)
			apiErr(w, err)
			return
		}
	}

	w.Write(routeCache)
}

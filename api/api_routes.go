package api

import (
	"log"
	"net/http"

	"github.com/brnstz/bus/internal/models"
)

type routesResp struct {
	Routes []*models.Route `json:"routes"`
}

/*
var (
	routeCache []byte
)

func InitRouteCache(agencyIDs []string) error {
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
*/

func GetRouteCache(regionID string) error {

}

func getRoutes(w http.ResponseWriter, r *http.Request) {
	var err error

	swlat, err := floatOrDie(r.FormValue("sw_lat"))
	if err != nil {
		apiErr(w, err)
		return
	}

	swlon, err := floatOrDie(r.FormValue("sw_lon"))
	if err != nil {
		apiErr(w, err)
		return
	}

	nelat, err := floatOrDie(r.FormValue("ne_lat"))
	if err != nil {
		apiErr(w, err)
		return
	}

	nelon, err := floatOrDie(r.FormValue("ne_lon"))
	if err != nil {
		apiErr(w, err)
		return
	}

	regionIDs, err := models.GetRegions(swlat, swlon, nelat, nelon)

	for _, v := range regionIDs {

	}

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

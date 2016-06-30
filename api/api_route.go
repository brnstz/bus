package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
)

type routesResp struct {
	Routes []*models.Route `json:"routes"`
}

func getRoutes(w http.ResponseWriter, r *http.Request) {
	var err error
	err = r.ParseForm()
	if err != nil {
		return
	}
	routes := []*models.Route{}

	agencyIDs := r.Form["agency_id"]
	routeIDs := r.Form["route_id"]

	if len(agencyIDs) > 0 && len(routeIDs) > 0 {
		// If agency_id and route_ids are passed, then assume
		// that's how we're querying
		if len(agencyIDs) != len(routeIDs) {
			err = errBadRequest
			return
		}

		for i := 0; i < len(routeIDs); i++ {
			route, err := models.GetRoute(agencyIDs[i], routeIDs[i], true)
			if err != nil {
				log.Println("can't get route", err)
				apiErr(w, err)
				return
			}
			routes = append(routes, route)
		}

	} else {
		// Otherwise they are querying by lat/lon

		lat, err := floatOrDie(r.FormValue("lat"))
		if err != nil {
			return
		}

		lon, err := floatOrDie(r.FormValue("lon"))
		if err != nil {
			return
		}

		meters, err := floatOrDie(r.FormValue("meters"))
		if err != nil {
			return
		}

		filter := r.FormValue("filter")

		sq := models.StopQuery{
			MidLat:     lat,
			MidLon:     lon,
			Dist:       meters,
			RouteType:  filter,
			Distinct:   true,
			Departures: false,
		}

		err = sq.Initialize()
		if err != nil {
			log.Println("can't initialize stop query", err)
			apiErr(w, err)
			return
		}

		stops, err := models.GetStopsByQuery(etc.DBConn, sq)
		if err != nil {
			log.Println("can't get stops", err)
			apiErr(w, err)
			return
		}

		// assumes agency_id + route_id is unique across agencies
		// ok for now until we build a routequery
		distinctRoutes := map[string]bool{}

		for _, v := range stops {
			if !distinctRoutes[v.AgencyID+v.RouteID] {
				route, err := models.GetRoute(v.AgencyID, v.RouteID, true)
				if err != nil {
					log.Println("can't get route", err)
					apiErr(w, err)
					return
				}
				routes = append(routes, route)
			}

			distinctRoutes[v.AgencyID+v.RouteID] = true
		}
	}

	rr := routesResp{
		Routes: routes,
	}

	b, err := json.Marshal(rr)
	if err != nil {
		log.Println("can't marshal to json", err)
		apiErr(w, err)
		return
	}

	w.Write(b)
}

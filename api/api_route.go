package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
)

func getRoute(w http.ResponseWriter, r *http.Request) {
	agencyID := r.FormValue("agency_id")
	routeID := r.FormValue("route_id")

	route, err := models.GetRoute(etc.DBConn, agencyID, routeID)
	if err != nil {
		log.Println("can't get route", err)
		apiErr(w, err)
		return
	}

	route.RouteShapes, err = models.GetSavedRouteShapes(
		etc.DBConn, route.AgencyID, route.RouteID,
	)
	if err != nil {
		log.Println("can't append shapes", err)
		return
	}

	b, err := json.Marshal(route)
	if err != nil {
		log.Println("can't marshal route to json", err)
		apiErr(w, err)
		return
	}

	w.Write(b)
}

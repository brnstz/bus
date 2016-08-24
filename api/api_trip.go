package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
)

func getTrip(w http.ResponseWriter, r *http.Request) {
	agencyID := r.FormValue("agency_id")
	routeID := r.FormValue("route_id")
	tripID := r.FormValue("trip_id")
	fallbackTripID := r.FormValue("fallback_trip_id")

	trip, err := models.ReallyGetTrip(etc.DBConn, agencyID, routeID, tripID, fallbackTripID, true)
	if err != nil {
		log.Println("can't get trip", err)
		apiErr(w, err)
		return
	}

	b, err := json.Marshal(trip)
	if err != nil {
		log.Println("can't marshal to json", err)
		apiErr(w, err)
		return
	}

	w.Write(b)
}

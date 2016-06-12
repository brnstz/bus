package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/brnstz/bus/internal/models"
	"github.com/julienschmidt/httprouter"
)

func getTrip(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// FIXME: hack
	agencyID := strings.Replace(p.ByName("agencyID"), "+", " ", -1)
	routeID := strings.Replace(p.ByName("routeID"), "+", " ", -1)
	tripID := strings.Replace(p.ByName("tripID"), "+", " ", -1)

	trip, err := models.GetTrip(agencyID, routeID, tripID)
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

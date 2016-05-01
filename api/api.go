package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
)

func NewHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v2/stops", getStops)
	mux.Handle("/", http.FileServer(http.Dir(conf.API.WebDir)))

	return mux
}

func floatOrDie(w http.ResponseWriter, r *http.Request, name string) (f float64, err error) {

	val := r.FormValue(name)
	f, err = strconv.ParseFloat(val, 64)
	if err != nil {
		log.Println("bad float value", val, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	return
}

// stopResponse is the value returned by getStops
type stopResponse struct {
	Results []stopResult `json:"results"`
}

type stopResult struct {
	Route      *models.Route `json:"route"`
	Stop       *models.Stop  `json:"stop"`
	Departures struct {
		Live      []*models.Departure `json:"live"`
		Scheduled []*models.Departure `json:"scheduled"`
	} `json:"departures"`
	Dist float64 `json:"dist"`
}

func getStops(w http.ResponseWriter, r *http.Request) {
	var err error

	lat, err := floatOrDie(w, r, "lat")
	if err != nil {
		return
	}

	lon, err := floatOrDie(w, r, "lon")
	if err != nil {
		return
	}

	miles, err := floatOrDie(w, r, "miles")
	if err != nil {
		return
	}

	filter := r.FormValue("filter")

	meters := etc.MileToMeter(miles)

	stops, err := models.GetStopsByLoc(etc.DBConn, lat, lon, meters, filter)
	if err != nil {
		log.Println("can't get stops", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := stopResponse{}
	resp.Results = make([]stopResult, len(stops))

	for i, stop := range stops {
		resp.Results[i].Route, err = models.GetRoute(stop.RouteID)
		if err != nil {
			log.Println("can't get route for stop", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp.Results[i].Stop = stop
		resp.Results[i].Dist = stop.Dist
		resp.Results[i].Departures.Live = stop.Live
		resp.Results[i].Departures.Scheduled = stop.Scheduled
	}

	b, err := json.Marshal(resp)
	if err != nil {
		log.Println("can't marshal to json", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(b)
}

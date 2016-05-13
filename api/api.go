package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
)

var (
	errBadRequest = errors.New("bad request")

	// errCodes is a mapping from known errors to HTTP status codes
	errCodes = map[error]int{
		models.ErrNotFound: http.StatusNotFound,
		errBadRequest:      http.StatusBadRequest,
	}
)

func NewHandler() http.Handler {
	mux := httprouter.New()

	mux.GET("/api/v2/stops", getStops)
	mux.GET("/api/v2/agencies/:agencyID/trips/:tripID", getTrip)

	mux.Handler("GET", "/", http.FileServer(http.Dir(conf.API.WebDir)))

	return mux
}

// stopResponse is the value returned by getStops
type stopResponse struct {
	UpdatedAt time.Time    `json:"updated_at"`
	Results   []stopResult `json:"results"`
}

type stopResult struct {
	// ID is a unique ID for this result. For now it is
	// "{route_id}_{stop_id}"
	ID         string        `json:"id"`
	Route      *models.Route `json:"route"`
	Stop       *models.Stop  `json:"stop"`
	Departures struct {
		Live      []*models.Departure `json:"live"`
		Scheduled []*models.Departure `json:"scheduled"`
	} `json:"departures"`
	Dist float64 `json:"dist"`
}

func newStopResponse() stopResponse {
	return stopResponse{
		UpdatedAt: time.Now().UTC(),
	}
}

func getStops(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
		apiErr(w, err)
		return
	}

	resp := newStopResponse()
	resp.Results = make([]stopResult, len(stops))

	for i, stop := range stops {
		resp.Results[i].Route, err = models.GetRoute(stop.RouteID)
		if err != nil {
			log.Println("can't get route for stop", err)
			apiErr(w, err)
			return
		}

		resp.Results[i].Stop = stop
		resp.Results[i].Dist = stop.Dist
		resp.Results[i].Departures.Live = stop.Live
		resp.Results[i].Departures.Scheduled = stop.Scheduled

		resp.Results[i].ID = resp.Results[i].Route.ID + "_" + resp.Results[i].Stop.ID
	}

	b, err := json.Marshal(resp)
	if err != nil {
		log.Println("can't marshal to json", err)
		apiErr(w, err)
		return
	}

	w.Write(b)
}

func getTrip(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// FIXME: hack
	agencyID := strings.Replace(p.ByName("agencyID"), "+", " ", -1)
	tripID := strings.Replace(p.ByName("tripID"), "+", " ", -1)

	trip, err := models.GetTrip(agencyID, tripID)
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

func floatOrDie(w http.ResponseWriter, r *http.Request, name string) (f float64, err error) {

	val := r.FormValue(name)
	f, err = strconv.ParseFloat(val, 64)
	if err != nil {
		log.Println("bad float value", val, err)
		err = errBadRequest
		return
	}

	return
}

// apiErr writes an appropriate response to w given the incoming error
// by looking at the errCodes map
func apiErr(w http.ResponseWriter, err error) {
	code, ok := errCodes[err]
	if !ok {
		code = http.StatusInternalServerError
	}

	w.WriteHeader(code)
}

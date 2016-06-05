package api

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
	"github.com/julienschmidt/httprouter"
)

var (
	errBadRequest = errors.New("bad request")

	// errCodes is a mapping from known errors to HTTP status codes
	errCodes = map[error]int{
		models.ErrNotFound:         http.StatusNotFound,
		models.ErrInvalidRouteType: http.StatusBadRequest,
		errBadRequest:              http.StatusBadRequest,
	}

	staticPaths = []string{"js", "css"}
)

func NewHandler() http.Handler {
	// Create our mux
	mux := httprouter.New()

	// Set up index and dynamic endpoints
	mux.GET("/", getIndex)
	mux.GET("/api/stops", getStops)
	mux.GET("/api/routes", getRoutes)
	mux.GET("/api/agencies/:agencyID/routes/:routeID/trips/:tripID", getTrip)

	// Create static endpoints
	for _, v := range staticPaths {
		endpoint := "/" + v + "/*filepath"
		dir := http.Dir(path.Join(conf.API.WebDir, v))
		mux.ServeFiles(endpoint, dir)
	}

	return mux
}

func getIndex(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	// Create index template
	indexTemplate, err := template.ParseFiles(
		path.Join(conf.API.WebDir, "index.html"),
	)
	if err != nil {
		apiErr(w, err)
	}

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	indexTemplate.Execute(w, map[string]interface{}{
		"BuildTimestamp": conf.API.BuildTimestamp,
	})
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

	DisplayTrip *models.Trip `json:"display_trip"`
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

	sq := models.StopQuery{
		MidLat:     lat,
		MidLon:     lon,
		Dist:       meters,
		RouteType:  filter,
		Departures: true,
		Distinct:   true,
	}
	err = sq.Initialize()
	if err != nil {
		log.Println("can't init stop query", err)
		apiErr(w, err)
		return
	}

	stops, err := models.GetStopsByQuery(etc.DBConn, sq)
	if err != nil {
		log.Println("can't get stops", err)
		apiErr(w, err)
		return
	}

	resp := newStopResponse()
	resp.Results = make([]stopResult, len(stops))

	for i, stop := range stops {
		resp.Results[i].Route, err = models.GetRoute(stop.AgencyID, stop.RouteID, false)
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

		// Get the most relevant trip for DisplayTrip
		departures := []*models.Departure{}
		departures = append(departures, stop.Live...)
		departures = append(departures, stop.Scheduled...)

		for _, v := range departures {
			t, err := models.GetTrip(
				resp.Results[i].Stop.AgencyID,
				resp.Results[i].Stop.RouteID,
				v.TripID,
			)
			if err != nil {
				log.Println("can't get display trip", err)
				continue
			}

			// Get the first one, then stop
			resp.Results[i].DisplayTrip = &t
			break
		}

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

type routesResp struct {
	Routes []*models.Route
}

func getRoutes(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

	sq := models.StopQuery{
		MidLat:     lat,
		MidLon:     lon,
		Dist:       meters,
		RouteType:  filter,
		Departures: false,
		Distinct:   true,
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

	routes := []*models.Route{}

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

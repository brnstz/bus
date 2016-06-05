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
	Stops []*models.Stop `json:"stops"`
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

	resp := stopResponse{
		Stops: stops,
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
	Routes []*models.Route `json:"routes"`
}

func getRoutes(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

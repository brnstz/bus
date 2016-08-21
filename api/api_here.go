package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/willf/bloom"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/fuse"
	"github.com/brnstz/bus/internal/models"
	"github.com/brnstz/bus/internal/partners"
)

var (
	// Formula for determining m and k values: http://hur.st/bloomfilter
	// n = approx number of items to insert
	// p = desired false positive rate (between 0 and 1)
	// m = ceil((n * log(p)) / log(1.0 / (pow(2.0, log(2.0)))))
	// k = round(log(2.0) * m / n)
	// with n = 300 and p = 0.001
	bloomM uint = 4314
	bloomK uint = 10

	minFirstDepartureTime = time.Duration(2) * time.Hour
)

// hereResponse is the value returned by getHere
type hereResponse struct {
	Stops  []*models.Stop     `json:"stops"`
	Routes []*models.Route    `json:"routes"`
	Trips  []*models.Trip     `json:"trips"`
	Filter *bloom.BloomFilter `json:"filter"`
}

func getHere(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp hereResponse
	var routes []*models.Route
	var now time.Time

	if len(r.FormValue("now")) > 0 {
		now, err = time.ParseInLocation("2006-01-02 15:04:05", r.FormValue("now"), time.Local)
		if err != nil {
			log.Println("can't parse time", err)
			apiErr(w, errBadRequest)
			return
		}
	} else {
		now = time.Now()
	}

	// Read values incoming from http request
	lat, err := floatOrDie(r.FormValue("lat"))
	if err != nil {
		apiErr(w, err)
		return
	}

	lon, err := floatOrDie(r.FormValue("lon"))
	if err != nil {
		apiErr(w, err)
		return
	}

	SWLat, err := floatOrDie(r.FormValue("sw_lat"))
	if err != nil {
		apiErr(w, err)
		return
	}

	SWLon, err := floatOrDie(r.FormValue("sw_lon"))
	if err != nil {
		apiErr(w, err)
		return
	}

	NELat, err := floatOrDie(r.FormValue("ne_lat"))
	if err != nil {
		apiErr(w, err)
		return
	}

	NELon, err := floatOrDie(r.FormValue("ne_lon"))
	if err != nil {
		apiErr(w, err)
		return
	}

	includeRoutes, err := boolOrDie(r.FormValue("routes"))
	if err != nil {
		apiErr(w, err)
		return
	}

	includeTrips, err := boolOrDie(r.FormValue("trips"))
	if err != nil {
		apiErr(w, err)
		return
	}

	// Initialize or read incoming bloom filter
	filter := r.FormValue("filter")

	var routeTypes []int
	for _, v := range r.Form["route_type"] {
		intv, err := strconv.Atoi(v)
		if err != nil {
			apiErr(w, errBadRequest)
		}

		routeTypes = append(routeTypes, intv)
	}

	if len(filter) < 1 {
		// If there is no filter, then create a new one
		resp.Filter = bloom.New(bloomM, bloomK)

	} else {
		resp.Filter = &bloom.BloomFilter{}
		// Otherwise read the passed value as JSON string
		err = json.Unmarshal([]byte(filter), resp.Filter)
		if err != nil {
			log.Println("can't read incoming bloom filter JSON", err)
			apiErr(w, errBadRequest)
			return
		}
	}

	hq, err := models.NewHereQuery(
		lat, lon, SWLat, SWLon, NELat, NELon, routeTypes, now,
	)
	if err != nil {
		log.Println("can't create here query", err)
		apiErr(w, err)
		return
	}

	stops, stopRoutes, err := models.GetHereResults(etc.DBConn, hq)
	if err != nil {
		log.Println("can't get here results", err)
		apiErr(w, err)
		return
	}

	// time taken to add stuff
	if conf.API.LogTiming {
		t1 := time.Now()
		defer func() { log.Println(time.Now().Sub(t1)) }()
	}

	// Create a channel for waiting for responses, arbitrarily large
	respch := make(chan error, 10000)
	count := 0

	// save the first scheduled departure of each stop, so that we can
	// use it in case the live tripID cannot be found
	firstDepart := map[string]*models.Departure{}

	var partner partners.P
	for _, s := range stops {

		firstDepart[s.UniqueID] = s.Departures[0]
		route := stopRoutes[s.UniqueID]
		routes = append(routes, route)

		// Get a live partner or skip it
		partner, err = partners.Find(*route)
		if err != nil {
			log.Println(err)
			continue
		}

		// Create a request to get live info and send it on the channel
		req := &fuse.StopReq{
			Stop:     s,
			Partner:  partner,
			Response: respch,
		}
		fuse.StopChan <- req
		count++
	}

	// Add any routes to the response that the bloom filter says we don't have
	for _, route := range routes {
		if !includeRoutes {
			break
		}

		exists := resp.Filter.TestString(route.UniqueID)
		if exists {
			continue
		}

		req := &fuse.RouteReq{
			Route:    route,
			Response: respch,
		}

		fuse.RouteChan <- req
		count++
	}

	// Set stop value of the response
	resp.Stops = stops

	// Add the first trip of each stop response that is not already in our
	// bloom filter
	tripReqs := []*fuse.TripReq{}
	for _, stop := range resp.Stops {

		// Even if we aren't including trips in response, we still
		// need to do this, but we can ignore non-live trips
		if !partner.IsLive() {
			continue
		}

		if len(stop.Departures) < 1 {
			continue
		}

		if now.Add(minFirstDepartureTime).Before(stop.Departures[0].Time) {
			continue
		}

		// Get info for the trip
		tripID := stop.Departures[0].TripID
		uniqueID := stop.AgencyID + "|" + tripID

		// Check if the trip already exists
		exists := resp.Filter.TestString(uniqueID)

		// If it exists, skip it
		if exists {
			continue
		}

		req := &fuse.TripReq{
			TripID:       tripID,
			FirstTripID:  firstDepart[stop.UniqueID].TripID,
			Stop:         stop,
			Response:     respch,
			IncludeShape: includeTrips,
		}
		tripReqs = append(tripReqs, req)
		fuse.TripChan <- req
		count++
	}

	// Wait for all responses
	for i := 0; i < count; i++ {
		err = <-respch
		if err != nil {
			log.Println(err)
		}
	}

	// append routes
	for _, route := range routes {
		if !includeRoutes {
			break
		}

		exists := resp.Filter.TestString(route.UniqueID)
		if exists {
			continue
		}

		resp.Filter.AddString(route.UniqueID)
		resp.Routes = append(resp.Routes, route)
	}

	// append trips
	for _, tripReq := range tripReqs {

		if tripReq.Trip == nil {
			continue
		}

		uniqueID := tripReq.Stop.AgencyID + "|" + tripReq.Trip.TripID

		exists := resp.Filter.TestString(uniqueID)
		if exists {
			continue
		}

		if tripReq.Stop.Departures[0].TripID != tripReq.Trip.TripID {
			log.Printf("switching from %v to %v", tripReq.Stop.Departures[0].TripID,
				tripReq.Trip.TripID)
			tripReq.Stop.Departures[0].TripID = tripReq.Trip.TripID
			// FIXME: do we need to do this? should this actually be to init
			// the departure?
			tripReq.Stop.Initialize()
		}

		// Only put trips in response if requested
		if includeTrips {
			resp.Trips = append(resp.Trips, tripReq.Trip)
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

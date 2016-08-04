package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/willf/bloom"

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

	t1 := time.Now()
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
	log.Println("here query:        ", time.Now().Sub(t1))

	t2 := time.Now()
	// Create a channel for receiving responses to stopLiveRequest values
	respch := make(chan error, len(stops))
	count := 0

	// save the first scheduled departure of each stop, so that we can
	// use it in case the live tripID cannot be found
	firstDepart := map[string]*models.Departure{}

	for _, s := range stops {

		firstDepart[s.UniqueID] = s.Departures[0]
		route := stopRoutes[s.UniqueID]
		routes = append(routes, route)

		// Get a live partner or skip it
		partner, err := partners.Find(*route)
		if err != nil {
			log.Println(err)
			continue
		}

		// Create a request to get live info and send it on the channel
		req := &fuse.Req{
			Stop:     s,
			Partner:  partner,
			Response: respch,
			Filter:   resp.Filter,
		}
		fuse.Chan <- req
		count++
	}

	// Wait for all responses
	for i := 0; i < count; i++ {
		err = <-respch
		if err != nil {
			log.Println(err)
		}
	}
	log.Println("partner info:      ", time.Now().Sub(t2))

	t3 := time.Now()
	// Add any routes to the response that the bloom filter says we don't have
	for _, route := range routes {
		exists := resp.Filter.TestString(route.UniqueID)
		// If the route doesn't exist in our filter, then we want to pull
		// the shapes and also append it to our response list.
		if !exists {
			route.RouteShapes, err = models.GetSavedRouteShapes(
				etc.DBConn, route.AgencyID, route.RouteID,
			)
			if err != nil {
				// This is a fatal error because the front end code
				// assumes the route will be there
				log.Println("can't get route shapes", route, err)
				apiErr(w, err)
				return
			}

			resp.Filter.AddString(route.UniqueID)
			resp.Routes = append(resp.Routes, route)
		}
	}
	log.Println("route shape:       ", time.Now().Sub(t3))

	t4 := time.Now()

	// Set stop value of the response
	resp.Stops = stops

	// Add the first trip of each stop response that is not already in our
	// bloom filter
	for i, stop := range resp.Stops {
		var trip models.Trip

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

		// Get the full trip with stop and shape details. If we succeed, we can
		// move onto next trip
		trip, err = models.GetTrip(etc.DBConn, stop.AgencyID, stop.RouteID, tripID)
		if err == nil {
			resp.Filter.AddString(uniqueID)
			resp.Trips = append(resp.Trips, &trip)
			continue
		}

		// If the error is unexpected, we should error out immediately
		if err != models.ErrNotFound {
			log.Println("can't get trip", err)
			apiErr(w, err)
			return
		}

		// Here we weren't able to find the trip ID in the database. This is
		// typically due to a response from a realtime source which gives us
		// TripIDs that are not in the static feed or are partial matches.
		// Let's first look for a partial match. If that fails, let's just get
		// the use the first scheduled departure instead.

		// Checking for partial match.
		tripID, err = models.GetPartialTripIDMatch(
			etc.DBConn, stop.AgencyID, stop.RouteID, tripID,
		)

		// If we get one, then update the uniqueID and the relevant stop /
		// departure's ID, adding it to our filter.
		if err == nil {
			uniqueID = stop.AgencyID + "|" + tripID
			resp.Stops[i].Departures[0].TripID = tripID
			resp.Stops[i].Initialize()

			// Re-get the trip with update ID
			trip, err = models.GetTrip(etc.DBConn, stop.AgencyID, stop.RouteID,
				tripID)
			if err != nil {
				log.Println("can't get trip", err)
				apiErr(w, err)
				return
			}

			resp.Filter.AddString(uniqueID)
			resp.Trips = append(resp.Trips, &trip)

			continue
		}

		// If the error is unexpected, we should error out immediately
		if err != models.ErrNotFound {
			log.Println("can't get trip", err)
			apiErr(w, err)
			return
		}

		// Our last hope is take the first scheduled departure
		tripID = firstDepart[stop.UniqueID].TripID

		uniqueID = stop.AgencyID + "|" + tripID
		resp.Stops[i].Departures[0].TripID = tripID
		resp.Stops[i].Initialize()

		// Re-get the trip with update ID
		trip, err = models.GetTrip(etc.DBConn, stop.AgencyID, stop.RouteID,
			tripID)
		if err != nil {
			log.Println("can't get trip", err)
			apiErr(w, err)
			return
		}

		resp.Filter.AddString(uniqueID)
		resp.Trips = append(resp.Trips, &trip)
	}
	log.Println("trips:             ", time.Now().Sub(t4))

	b, err := json.Marshal(resp)
	if err != nil {
		log.Println("can't marshal to json", err)
		apiErr(w, err)
		return
	}

	w.Write(b)
}

package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/willf/bloom"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
	"github.com/brnstz/bus/internal/partners"
)

var (
	// stopChan is a channel for receiving requests to get live departure
	// data
	stopChan chan *stopLiveRequest

	// workers is the number of workers processing requestChan concurrently
	stopWorkers = 10

	// Formula for determining m and k values: http://hur.st/bloomfilter
	// n = approx number of items to insert
	// p = desired false positive rate (between 0 and 1)
	// m = ceil((n * log(p)) / log(1.0 / (pow(2.0, log(2.0)))))
	// k = round(log(2.0) * m / n)
	// with n = 300 and p = 0.001
	bloomM uint = 4314
	bloomK uint = 10
)

type stopLiveRequest struct {
	route    *models.Route
	stop     *models.Stop
	partner  partners.P
	response chan error
}

func init() {
	stopChan = make(chan *stopLiveRequest, 100000)

	for i := 0; i < stopWorkers; i++ {
		go stopWorker()
	}
}

// stop worker calls the partner's live departure API and sets
// req.stop.Live
func stopWorker() {
	for req := range stopChan {
		liveDepartures, liveVehicles, err := req.partner.Live(req.route.AgencyID, req.route.RouteID, req.stop.StopID, req.stop.DirectionID)
		if err != nil {
			req.response <- err
			continue
		}

		if len(liveVehicles) > 0 {
			req.stop.Vehicles = liveVehicles
		}

		// Ensure the departures are sorted
		sort.Sort(liveDepartures)

		if len(liveDepartures) > 0 {
			liveTripIDs := map[string]bool{}

			// Remove any of the same trip ids that appear in scheduled
			// departures. Live info is better for that trip, but there
			// might still be scheduled departures later we want to use.
			for _, d := range liveDepartures {
				liveTripIDs[d.TripID] = true
			}

			// If there are less than max departures, then add scheduled
			// departures that are after our last live departure and
			// don't have dupe trip IDs
			count := len(liveDepartures)
			lastLiveDeparture := liveDepartures[count-1]

			i := -1
			for {
				i++

				// Stop once we have enough departures
				if count >= models.MaxDepartures {
					break
				}

				// Stop if we reach the end of the scheduled departures
				if i >= len(req.stop.Departures) {
					break
				}

				// Ignore departures with trip IDs that we know of
				if liveTripIDs[req.stop.Departures[i].TripID] {
					continue
				}

				if req.stop.Departures[i].Time.After(lastLiveDeparture.Time) {
					liveDepartures = append(liveDepartures, req.stop.Departures[i])
				}

			}

			if len(liveDepartures) > 5 {
				req.stop.Departures = liveDepartures[0:5]
			} else {
				req.stop.Departures = liveDepartures
			}

		}

		req.response <- nil
	}
}

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

	now := time.Now()

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

	today := etc.BaseTime(now)
	todayName := strings.ToLower(now.Format("Monday"))
	// FIXME: hard coded, we need a region to agency mapping
	agencyID := "MTA NYCT"
	todayIDs, err := models.GetNewServiceIDs(etc.DBConn, agencyID, todayName, today)
	if err != nil {
		log.Println("can't get serviceIDs", err)
		apiErr(w, err)
		return
	}

	hq, err := models.NewHereQuery(
		lat, lon, SWLat, SWLon, NELat, NELon,
		todayIDs, etc.TimeToDepartureSecs(now), today,
	)

	stops, stopRoutes, err := models.GetHereResults(etc.DBConn, hq)
	if err != nil {
		log.Println("can't get here results", err)
		apiErr(w, err)
		return
	}

	// Create a channel for receiving responses to stopLiveRequest values
	respch := make(chan error, len(stops))
	count := 0

	// save the first scheduled departure of each stop, so that we can
	// use it in case the live tripID cannot be found
	firstDepart := map[string]*models.Departure{}

	t3 := time.Now()
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
		req := &stopLiveRequest{
			route:    route,
			stop:     s,
			partner:  partner,
			response: respch,
		}
		stopChan <- req
		count++
	}

	// Wait for all responses
	for i := 0; i < count; i++ {
		err = <-respch
		if err != nil {
			log.Println(err)
		}
	}

	// Set stop value of the response
	resp.Stops = stops

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
	log.Println("time spent getting routes and partners", time.Now().Sub(t3))

	t4 := time.Now()

	// Add the first trip of each stop response that is not already in our
	// bloom filter
	for i, stop := range stops {
		var trip models.Trip

		if len(stop.Departures) < 1 {
			continue
		}

		// Get info for the trip
		tripID := stop.Departures[0].TripID
		uniqueID := stop.AgencyID + "|" + tripID

		// Check if the trip already exists
		exists := resp.Filter.TestAndAddString(uniqueID)

		// If it exists, skip it
		if exists {
			continue
		}

		// Get the full trip with stop and shape details. If we succeed, we can
		// move onto next trip
		trip, err = models.GetTrip(etc.DBConn, stop.AgencyID, stop.RouteID, tripID)
		if err == nil {
			log.Println("got it normally", uniqueID, stop.RouteID, stop.Headsign)
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
			etc.DBConn, agencyID, stop.RouteID, tripID,
		)

		// If we get one, then update the uniqueID and the relevant stop /
		// departure's ID, adding it to our filter.
		if err == nil {
			uniqueID = stop.AgencyID + "|" + tripID
			resp.Stops[i].Departures[0].TripID = tripID

			// Re-get the trip with update ID
			trip, err = models.GetTrip(etc.DBConn, agencyID, stop.RouteID,
				tripID)
			if err != nil {
				log.Println("can't get trip", err)
				apiErr(w, err)
				return
			}

			log.Println("got it partial match", uniqueID, stop.RouteID, stop.Headsign)
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

		// Re-get the trip with update ID
		trip, err = models.GetTrip(etc.DBConn, agencyID, stop.RouteID,
			tripID)
		if err != nil {
			log.Println("can't get trip", err)
			apiErr(w, err)
			return
		}

		log.Println("got it first departure", uniqueID, stop.RouteID, stop.Headsign)
		resp.Filter.AddString(uniqueID)
		resp.Trips = append(resp.Trips, &trip)
	}
	log.Println("time spent getting trips", time.Now().Sub(t4))

	t5 := time.Now()

	b, err := json.Marshal(resp)
	if err != nil {
		log.Println("can't marshal to json", err)
		apiErr(w, err)
		return
	}
	log.Println("time spent marshalling", time.Now().Sub(t5))

	w.Write(b)

}

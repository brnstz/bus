package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
	"github.com/brnstz/bus/internal/partners"
	"github.com/willf/bloom"
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
		liveDepartures, liveVehicles, err := req.partner.Live(*req.route, *req.stop)
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
type stopResponse struct {
	Stops  []*models.Stop     `json:"stops"`
	Routes []*models.Route    `json:"routes"`
	Filter *bloom.BloomFilter `json:"filter"`
}

func getHere(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp stopResponse
	var routes []*models.Route

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
		// Otherwise read the passed value as JSON string
		err = json.Unmarshal([]byte(filter), resp.Filter)
		if err != nil {
			log.Println("can't read incoming bloom filter JSON", err)
			apiErr(w, errBadRequest)
			return
		}
	}

	// Create a query for stops
	sq := models.StopQuery{
		MidLat:     lat,
		MidLon:     lon,
		SWLat:      SWLat,
		SWLon:      SWLon,
		NELat:      NELat,
		NELon:      NELon,
		Distinct:   true,
		Departures: true,
	}
	err = sq.Initialize()
	if err != nil {
		log.Println("can't init stop query", err)
		apiErr(w, err)
		return
	}

	// Get stops that match this query
	stops, err := models.GetStopsByQuery(etc.DBConn, sq)
	if err != nil {
		log.Println("can't get stops", err)
		apiErr(w, err)
		return
	}

	// Create a channel for receiving responses to stopLiveRequest values
	respch := make(chan error, len(stops))
	count := 0

	for _, s := range stops {

		// Get the route for this stop and add to our list (may include dupes)
		route, err := models.GetRouteV2(etc.DBConn, s.AgencyID, s.RouteID)
		if err != nil {
			log.Println("can't get route", err)
			apiErr(w, err)
			return
		}
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
		exists := resp.Filter.TestAndAddString(route.UniqueID)
		// If the route doesn't exist in our filter, then we want to pull
		// the shapes and also append it to our response list.
		if !exists {
			route.RouteShapes, err = models.GetSavedRouteShapes(
				etc.DBConn, route.AgencyID, route.RouteID,
			)
			if err != nil {
				log.Println("can't get route shapes", err)
				apiErr(w, err)
				return
			}

			resp.Routes = append(resp.Routes, route)
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

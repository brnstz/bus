package fuse

import (
	"log"
	"sort"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
	"github.com/brnstz/bus/internal/partners"
)

var (
	StopChan  chan *StopReq
	RouteChan chan *RouteReq
	TripChan  chan *TripReq

	workers = 100
)

// StopReq is a request to set live departures for a stop using
// the Partner
type StopReq struct {
	Stop     *models.Stop
	Partner  partners.P
	Response chan error
}

// RouteReq is a request to add saved route shapes to this route
type RouteReq struct {
	Route    *models.Route
	Response chan error
}

// TripReq is a request to retrieve a Trip given this TripID and Stop
type TripReq struct {
	Trip         *models.Trip
	TripID       string
	FirstTripID  string
	Stop         *models.Stop
	Response     chan error
	IncludeShape bool
}

func init() {
	StopChan = make(chan *StopReq, 100000)
	RouteChan = make(chan *RouteReq, 100000)
	TripChan = make(chan *TripReq, 100000)

	for i := 0; i < workers; i++ {
		go stopWorker()
		go routeWorker()
		go tripWorker()
	}
}

// stopWorker calls the partner's live departure API and sets
// req.Stop.Live
func stopWorker() {
	for req := range StopChan {
		liveDepartures, liveVehicles, err := req.Partner.Live(req.Stop.AgencyID, req.Stop.RouteID, req.Stop.StopID, req.Stop.DirectionID)
		if err != nil {
			req.Response <- err
			continue
		}

		if len(liveVehicles) > 0 {
			req.Stop.Vehicles = liveVehicles
		}

		// FIXME: assume compass dir for live departures is
		// the first scheduled departure's dir
		compassDir := req.Stop.Departures[0].CompassDir

		sd := models.SortableDepartures(liveDepartures)
		sort.Sort(sd)
		liveDepartures = []*models.Departure(sd)

		if len(liveDepartures) > 0 {
			liveTripIDs := map[string]bool{}

			// Remove any of the same trip ids that appear in scheduled
			// departures. Live info is better for that trip, but there
			// might still be scheduled departures later we want to use.
			for _, d := range liveDepartures {
				liveTripIDs[d.TripID] = true
				d.CompassDir = compassDir
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
				if i >= len(req.Stop.Departures) {
					break
				}

				// Ignore departures with trip IDs that we know of
				if liveTripIDs[req.Stop.Departures[i].TripID] {
					continue
				}

				if req.Stop.Departures[i].Time.After(lastLiveDeparture.Time) {
					liveDepartures = append(liveDepartures, req.Stop.Departures[i])
				}

			}

			if len(liveDepartures) > models.MaxDepartures {
				req.Stop.Departures = liveDepartures[0:models.MaxDepartures]
			} else {
				req.Stop.Departures = liveDepartures
			}

		}

		req.Response <- nil
	}
}

func routeWorker() {
	var err error

	for req := range RouteChan {
		req.Route.RouteShapes, err = models.GetSavedRouteShapes(
			etc.DBConn, req.Route.AgencyID, req.Route.RouteID,
		)
		if err != nil {
			// This is a fatal error because the front end code
			// assumes the route will be there
			log.Println("can't get route shapes", req.Route, err)
			req.Response <- err
			continue
		}

		req.Response <- nil
	}
}

func tripWorker() {

	for req := range TripChan {
		var err error
		var trip *models.Trip

		trip, err = models.ReallyGetTrip(etc.DBConn, req.Stop.AgencyID,
			req.Stop.RouteID, req.TripID, req.FirstTripID, req.IncludeShape)

		if err != nil {
			req.Response <- err
		} else {
			req.Trip = trip
			req.Response <- nil
		}
	}
}

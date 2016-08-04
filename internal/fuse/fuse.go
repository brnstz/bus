package fuse

import (
	"sort"

	"github.com/brnstz/bus/internal/models"
	"github.com/brnstz/bus/internal/partners"
	"github.com/willf/bloom"
)

var (
	Chan chan *Req

	// workers is the number of workers processing requestChan concurrently
	workers = 10
)

type Req struct {
	Stop     *models.Stop // will be modified in place
	Route    *models.Route
	Trip     *models.Trip
	Filter   *bloom.BloomFilter
	Partner  partners.P
	Response chan error
}

func init() {
	Chan = make(chan *Req, 100000)

	for i := 0; i < workers; i++ {
		go worker()
	}
}

// worker calls the partner's live departure API and sets
// req.stop.Live
func worker() {
	for req := range Chan {
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

			for i := 0; i < models.MaxDepartures && i < len(liveDepartures); i++ {
				liveDepartures[i].CompassDir = compassDir
				req.Stop.Departures[i] = liveDepartures[i]

			}

		}

		req.Response <- nil
	}
}

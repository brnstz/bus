package partners

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"

	"github.com/brnstz/bus/internal/partners/nyct_subway"
	"github.com/brnstz/bus/internal/partners/transit_realtime"

	"github.com/golang/protobuf/proto"
)

var (
	esiURL = "http://datamine.mta.info/mta_esi.php"

	routeToFeed = map[string]string{
		"1":  "1",
		"2":  "1",
		"3":  "1",
		"4":  "1",
		"5":  "1",
		"6":  "1",
		"6X": "1",
		"S":  "1",
		"GS": "1",
		"L":  "2",
		"SI": "11",
	}
)

type mtaNYCSubway struct{}

func (_ mtaNYCSubway) Live(route models.Route, stop models.Stop) (d models.Departures, v []models.Vehicle, err error) {

	// Get the feed for this route, if there is one. Otherwise, nothing
	// to return.
	feed, exists := routeToFeed[stop.RouteID]
	if !exists {
		return
	}

	// Construct URL and call external API, possibly getting cached
	// value.
	q := url.Values{}
	q.Set("key", conf.API.DatamineAPIKey)
	q.Set("feed_id", feed)
	u := fmt.Sprint(esiURL, "?", q.Encode())

	b, err := etc.RedisCache(u)
	if err != nil {
		log.Println("can't get live subways", err)
		return
	}

	// Load the protobuf struct
	tr := &transit_realtime.FeedMessage{}
	err = proto.Unmarshal(b, tr)
	if err != nil {
		log.Println("can't unmarshal", err)
		return
	}

	// Look at each message in the feed
	for _, e := range tr.Entity {

		// Get some updates
		tripUpdate := e.GetTripUpdate()
		trip := tripUpdate.GetTrip()
		stopTimeUpdates := tripUpdate.GetStopTimeUpdate()

		// If we have at least one stopTimeUpdate and the trip is non-nil,
		// we can get the NYCT extensions.
		if len(stopTimeUpdates) > 0 && trip != nil {
			var event interface{}

			// Get the NYC extension so we can see if the Trip is "assigned"
			// yet If it's assigned, we'll put the vehicle on the map.
			event, err = proto.GetExtension(
				trip, nyct_subway.E_NyctTripDescriptor,
			)
			if err != nil {
				log.Println("can't get extension", err)
				return
			}
			nycTrip, ok := event.(*nyct_subway.NyctTripDescriptor)
			if !ok {
				log.Println("can't coerce to nyct_subway.NyctTripDescriptor")
				return
			}

			// The first update in an entity is the stop where the train will
			// next be. Include only "assigned" trips, which are those that
			// are about to start.
			if nycTrip.GetIsAssigned() {
				var vehicle models.Vehicle

				// Get a "vehicle" with the lat/lon of the update's stop
				// (*not* the stop of our request)
				vehicle, err = models.GetVehicle(
					route.AgencyID, route.RouteID,
					stopTimeUpdates[0].GetStopId(),
					stop.DirectionID,
				)
				if err != nil {
					// FIXME: identify not found error vs. others
					//log.Println("can't get vehicle", err)

				} else {
					vehicle.Live = true
					v = append(v, vehicle)
				}
			}
		}

		// Go through all updates to check for our stop ID's departure
		// time.
		for _, u := range stopTimeUpdates {
			// If this is our stop, then get the departure time.
			if u.GetStopId() == stop.ID {
				d = append(d,
					&models.Departure{
						Time:   time.Unix(u.GetDeparture().GetTime(), 0),
						TripID: trip.GetTripId(),
						Live:   true,
					},
				)
			}
		}
	}

	return
}

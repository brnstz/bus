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

	mtaSubwayRouteToFeed = map[string]string{
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

// getURL returns the URL for getting this routeID's feed. Second return
// value is false if there is no feed to get.
func (p mtaNYCSubway) getURL(routeID string) (string, bool) {
	var u string

	// Get the feed for this route, if there is one. Otherwise, nothing
	// to return.
	feed, exists := mtaSubwayRouteToFeed[routeID]
	if !exists {
		return "", false
	}

	// Construct URL and call external API, possibly getting cached
	// value.
	q := url.Values{}
	q.Set("key", conf.Partner.DatamineAPIKey)
	q.Set("feed_id", feed)
	u = fmt.Sprint(esiURL, "?", q.Encode())

	return u, true
}

func (p mtaNYCSubway) Precache(agencyID, routeID string, directionID int) error {
	u, exists := p.getURL(routeID)
	if !exists {
		return nil
	}

	// Since the URL we call is the same no matter which direction, arbirarily
	// decide to ignore one of the directions.
	if directionID == 1 {
		return nil
	}

	_, err := etc.RedisCacheURL(u)
	if err != nil {
		log.Println("can't cache live subway response", err)
		return err
	}

	// attempt to parse response to ensure it is valid
	_, _, err = p.Live(agencyID, routeID, "", directionID)
	if err != nil {
		log.Println("can't parse response", err)
		return err
	}

	return nil
}

func (p mtaNYCSubway) Live(agencyID, routeID, stopID string, directionID int) (d models.Departures, v []models.Vehicle, err error) {
	now := time.Now()

	u, exists := p.getURL(routeID)
	if !exists {
		return
	}

	b, err := etc.RedisGet(u)
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
			// yet. If it's assigned, we'll put the vehicle on the map.
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
					agencyID, routeID,
					stopTimeUpdates[0].GetStopId(),
					directionID,
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

		// FIXME: we should probably do this in api_here
		/*
			tripID, err := models.GetPartialTripIDMatch(etc.DBConn, agencyID, routeID, trip.GetTripId())
			if err != nil {
				// FIXME: what to do here?
				tripID = trip.GetTripId()
				log.Println("can't get tripID", routeID, tripID, trip.GetTripId(), err)
			}
		*/

		// Go through all updates to check for our stop ID's departure time.
		for _, u := range stopTimeUpdates {

			// If this is our stop, then get the departure time.
			if u.GetStopId() == stopID {
				dtime := time.Unix(u.GetDeparture().GetTime(), 0)
				if dtime.After(now) {
					d = append(d,
						&models.Departure{
							Time:   dtime,
							TripID: trip.GetTripId(),
							Live:   true,
						},
					)
				}
			}
		}
	}

	return
}

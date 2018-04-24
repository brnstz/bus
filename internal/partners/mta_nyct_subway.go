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
	cacheDirection = 0

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

		"L": "2",

		"SI": "11",

		"N": "16",
		"Q": "16",
		"R": "16",
		"W": "16",

		"B": "21",
		"D": "21",
		"F": "21",
		"M": "21",

		"A": "26",
		"C": "26",
		"E": "26",
		"H": "26",

		"G": "31",

		"J": "36",
		"Z": "36",

		"7": "51",
	}

	cacheRoute = map[string]bool{}
)

func init() {
	seenFeed := map[string]bool{}

	// Cache exactly one route per feed
	for route, feed := range mtaSubwayRouteToFeed {
		if seenFeed[feed] {
			continue
		}

		cacheRoute[route] = true
		seenFeed[feed] = true
	}

}

type mtaNYCSubway struct{}

func (p mtaNYCSubway) IsLive() bool {
	return true
}

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
	k := fmt.Sprintf("%v|%v|%v", agencyID, routeID, directionID)

	u, exists := p.getURL(routeID)
	if !exists {
		return nil
	}

	// Since the URL we call is the same no matter which direction, arbirarily
	// decide to ignore one of the directions.
	if directionID != cacheDirection {
		return nil
	}

	// Since multiple routes appear in the same feed, only need to cache
	// one route.
	if !cacheRoute[mtaSubwayRouteToFeed[routeID]] {
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

	log.Println("succesfully saved", k)

	return nil
}

func (p mtaNYCSubway) Live(agencyID, routeID, stopID string, directionID int) (d []*models.Departure, v []models.Vehicle, err error) {
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

		// Ensure we have at least one stop time update
		if len(stopTimeUpdates) < 1 {
			continue
		}

		// Check the first stop time update to see if it's express or not
		var updateEvent interface{}
		firstUpdate := stopTimeUpdates[0]

		// Get the NYC extension so we can see if the Trip is "assigned"
		// yet. If it's assigned, we'll put the vehicle on the map.
		updateEvent, err = proto.GetExtension(
			firstUpdate, nyct_subway.E_NyctStopTimeUpdate,
		)
		if err != nil {
			log.Println("can't get extension", err)
			return
		}
		nycEvent, ok := updateEvent.(*nyct_subway.NyctStopTimeUpdate)
		if !ok {
			log.Println("can't coerce to nyct_subway.NyctStopTimeUpdate")
			return
		}

		var feedRouteID string

		switch nycEvent.GetScheduledTrack() {

		case "2", "3", "M":
			// Express track. Special case for 6X.
			if trip.GetRouteId() == "6" {
				feedRouteID = trip.GetRouteId() + "X"
			} else {
				feedRouteID = trip.GetRouteId()
			}
		default:
			// not express track
			feedRouteID = trip.GetRouteId()
		}

		if feedRouteID != routeID {
			continue
		}

		// If we have at least one stopTimeUpdate (already checked before) and
		// the trip is non-nil, we can get the NYCT extensions.
		if trip != nil {
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
					// FIXME: why are these showing up again?
					//log.Println("can't get vehicle", err)

				} else {
					vehicle.Live = true
					v = append(v, vehicle)
				}
			}
		}

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

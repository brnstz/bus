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

func init() {
	log.Println("hello", nyct_subway.NyctTripDescriptor_Direction_name[1])
}

type mtaNYCSubway struct{}

func (_ mtaNYCSubway) Live(route models.Route, stop models.Stop) (d models.Departures, v []models.Vehicle, err error) {
	//now := time.Now()

	feed, exists := routeToFeed[stop.RouteID]
	if !exists {
		return
	}

	q := url.Values{}
	q.Set("key", conf.API.DatamineAPIKey)
	q.Set("feed_id", feed)
	u := fmt.Sprint(esiURL, "?", q.Encode())

	b, err := etc.RedisCache(u)
	if err != nil {
		log.Println("can't get live subways", err)
		return
	}

	tr := &transit_realtime.FeedMessage{}
	err = proto.Unmarshal(b, tr)
	if err != nil {
		log.Println("can't unmarshal", err)
		return
	}

	for _, e := range tr.Entity {
		//log.Println("what is it?", e)
		var vehicle models.Vehicle

		tripUpdate := e.GetTripUpdate()
		log.Println("what is trip?", tripUpdate.GetTrip().String())

		tripID := tripUpdate.GetTrip().GetTripId()

		updates := tripUpdate.GetStopTimeUpdate()

		first := true
		for _, u := range updates {
			//log.Println("what is update?", u)

			stopID := u.GetStopId()
			departureTime := time.Unix(u.GetDeparture().GetTime(), 0)

			// The first update in an entity is the stop where the train will
			// next be (including trips that haven't started yet)
			if first {
				first = false

				vehicle, err = models.GetVehicle(route.AgencyID, route.ID, stop.ID)
				vehicle.Live = true
				if err != nil {
					log.Println("can't get vehicle", err)
					return
				}
				v = append(v, vehicle)
			}

			// If this is our stop, then get the departure time.
			if stopID == stop.ID {
				d = append(d,
					&models.Departure{
						Time:   departureTime,
						TripID: tripID,
						Live:   true,
					},
				)
			}
		}
	}

	return
}

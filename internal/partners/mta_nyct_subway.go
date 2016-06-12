package partners

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
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

func (_ mtaNYCSubway) LiveDepartures(route models.Route, stop models.Stop) (d models.Departures, err error) {

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
	}

	for _, e := range tr.Entity {
		tripID := e.TripUpdate.GetTrip().GetTripId()
		updates := e.GetTripUpdate().GetStopTimeUpdate()
		for _, u := range updates {
			if u.GetStopId() == stop.ID {
				d = append(d,
					&models.Departure{
						Time:   time.Unix(u.GetDeparture().GetTime(), 0),
						TripID: tripID,
					},
				)
			}
		}
	}

	return

}

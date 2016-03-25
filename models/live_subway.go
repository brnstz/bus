package models

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/brnstz/bus/common"
	"github.com/brnstz/bus/transit_realtime"
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

func GetLiveSubways(route, dir, stop string) (ts timeSlice, err error) {
	feed, exists := routeToFeed[route]
	if !exists {
		return
	}

	q := url.Values{}
	q.Set("key", common.SubwayAPIKey)
	q.Set("feed_id", feed)
	u := fmt.Sprint(esiURL, "?", q.Encode())

	b, err := common.RedisCache(u)
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
		updates := e.GetTripUpdate().GetStopTimeUpdate()
		for _, u := range updates {
			if u.GetStopId() == stop {
				ts = append(ts, time.Unix(u.GetDeparture().GetTime(), 0))
			}
		}
	}

	return
}

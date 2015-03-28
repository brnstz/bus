package models

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/brnstz/bus/transit_realtime"
	"github.com/golang/protobuf/proto"
)

var (
	subwayKey = os.Getenv("MTA_SUBWAY_TIME_API_KEY")
	esiURL    = "http://datamine.mta.info/mta_esi.php"

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
	q.Set("key", subwayKey)
	q.Set("feed_id", feed)
	u := fmt.Sprint(esiURL, "?", q.Encode())

	log.Println(u)

	resp, err := http.Get(u)
	if err != nil {
		log.Println("can't get live feed", err)
		return
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("can't read body", err)
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
				ts = append(ts, time.Unix(u.GetArrival().GetTime(), 0))
			}
		}
	}

	return
}

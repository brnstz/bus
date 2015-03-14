package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/golang/protobuf/proto"

	//			"github.com/golang/protobuf/proto"
	"github.com/brnstz/mta/transit_realtime"
)

var (
	mtaRealtimeURL   = "http://datamine.mta.info/mta_esi.php?key=%s&feed_id=%d"
	mtaRealtimeKey   = os.Getenv("MTA_REALTIME_API_KEY")
	mtaRealtimeFeeds = []int{1, 2}
)

func getAll() {
	for _, id := range mtaRealtimeFeeds {
		getOne(id)
	}
}

func getOne(id int) (err error) {

	/*
		resp, err := http.Get(fmt.Sprintf(mtaRealtimeURL, mtaRealtimeKey, id))
		if err != nil {
			log.Println("can't get realtime feed", err)
			return
		}
		defer resp.Body.Close()

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("can't read body", err)
			return
		}*/

	//b, err := ioutil.ReadFile("ltrain.gtfs")
	b, err := ioutil.ReadFile("othertrain.gtfs")
	if err != nil {
		log.Println("can't read body", err)
		return
	}

	tr := &transit_realtime.FeedMessage{}
	err = proto.Unmarshal(b, tr)
	if err != nil {
		log.Println("can't unmarshal", err)
	}

	// success

	log.Println(tr)

	return
}

func main() {
	getAll()
}

package main

import (
	"log"
	"strings"

	"github.com/brnstz/bus/common"
	"github.com/brnstz/bus/loader"
	_ "github.com/lib/pq"
)

func main() {
	db := common.DB

	for _, dir := range []string{
		"/Users/bseitz/go/src/github.com/brnstz/bus/schema/subway/",
		"/Users/bseitz/go/src/github.com/brnstz/bus/schema/brooklyn/",
		"/Users/bseitz/go/src/github.com/brnstz/bus/schema/manhattan/",
		"/Users/bseitz/go/src/github.com/brnstz/bus/schema/queens/",
		"/Users/bseitz/go/src/github.com/brnstz/bus/schema/staten_island/",
		"/Users/bseitz/go/src/github.com/brnstz/bus/schema/bronx/",
	} {

		stype := ""
		if strings.HasSuffix(dir, "subway/") {
			stype = "subway"
		} else {
			stype = "bus"
		}

		log.Println(dir)
		l := loader.NewLoader(dir)

		for _, s := range l.Stops {
			log.Println("Inserting stop: ", s)
			_, err := db.Exec(`
				INSERT INTO stop
				(stop_id, stop_name, direction_id, headsign, route_id,
				 location, stype)
				VALUES($1, $2, $3, $4, $5, ll_to_earth($6, $7), $8)
			`,
				s.Id, s.Name, s.DirectionId, s.Headsign, s.RouteId,
				s.Lat, s.Lon, stype,
			)

			if err != nil {
				log.Fatal(err)
			}
		}

		for _, s := range l.ScheduledStopTimes {
			log.Println("Inserting scheduled stop time: ", s)
			_, err := db.Exec(`
				INSERT INTO scheduled_stop_time
				(route_id, stop_id, service_id, departure_sec)
				VALUES($1, $2, $3, $4)
			`, s.RouteId, s.StopId, s.ServiceId, s.DepartureSec,
			)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

}

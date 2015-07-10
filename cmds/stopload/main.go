package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"time"

	"github.com/brnstz/bus/common"
	"github.com/brnstz/bus/loader"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func doOne(dir string, db *sqlx.DB) {

	stype := ""
	if strings.HasSuffix(dir, "subway/") {
		stype = "subway"
	} else {
		stype = "bus"
	}

	log.Println(dir, stype)
	l := loader.NewLoader(dir)

	for _, s := range l.ServiceRouteDays {
		log.Println("Inserting service route day: ", s)
		_, err := db.Exec(`
				INSERT INTO service_route_day
				(route_id, service_id, day, start_date, end_date)
				VALUES($1, $2, $3, $4, $5)
			`, s.RouteId, s.ServiceId, s.Day, s.StartDate, s.EndDate,
		)
		if err != nil {
			log.Println("ERROR SERVICE ROUTE DAYS: ", err, s)
		}
	}

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
			log.Println("ERROR STOPS: ", err, s)
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
			log.Println("ERROR SCHEDULED STOP TIMES: ", err, s)
		}
	}
}

func main() {
	db := common.DB
	root := flag.String("dir", "", "directory with extracted mta data files")
	if root == nil {
		panic("must provide -dir")
	}

	// Dump stack trace on kill
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Println("got signal", sig)
			panic("stack trace here")
		}
	}()

	for _, subdir := range []string{
		"subway/",
		"brooklyn/",
		"manhattan/",
		"queens/",
		"staten_island/",
		"bronx/",
	} {
		dir := path.Join(root, subdir)

		t1 := time.Now()
		doOne(dir, db)
		t2 := time.Now()
		log.Printf("took %v for %v\n", t2.Sub(t1), dir)
	}

	log.Println("finished all boroughs")
}

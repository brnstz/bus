package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"time"

	"github.com/brnstz/bus/common"
	"github.com/brnstz/bus/loader"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func doOne(dir string, stype string, db *sqlx.DB) {

	log.Println(dir, stype)
	l := loader.NewLoader(dir)

	for i, s := range l.ServiceRouteDays {
		_, err := db.Exec(`
				INSERT INTO service_route_day
				(route_id, service_id, day, start_date, end_date)
				VALUES($1, $2, $3, $4, $5)
			`, s.RouteId, s.ServiceId, s.Day, s.StartDate, s.EndDate,
		)
		if err != nil && !strings.Contains(err.Error(), "violates unique constraint") {
			log.Println("ERROR SERVICE ROUTE DAYS: ", err, s)
		}

		if i%100 == 0 {
			log.Printf("loaded %v service route days", i)
		}

	}

	for i, s := range l.Stops {
		_, err := db.Exec(`
				INSERT INTO stop
				(stop_id, stop_name, direction_id, headsign, route_id,
				 location, stype)
				VALUES($1, $2, $3, $4, $5, ll_to_earth($6, $7), $8)
			`,
			s.Id, s.Name, s.DirectionId, s.Headsign, s.RouteId,
			s.Lat, s.Lon, stype,
		)

		if err != nil && !strings.Contains(err.Error(), "violates unique constraint") {
			log.Println("ERROR STOPS: ", err, s)
		}

		if i%100 == 0 {
			log.Printf("loaded %v stops", i)
		}
	}

	for i, s := range l.ScheduledStopTimes {
		_, err := db.Exec(`
				INSERT INTO scheduled_stop_time
				(route_id, stop_id, service_id, departure_sec)
				VALUES($1, $2, $3, $4)
			`, s.RouteId, s.StopId, s.ServiceId, s.DepartureSec,
		)
		if err != nil && !strings.Contains(err.Error(), "violates unique constraint") {
			log.Println("ERROR SCHEDULED STOP TIMES: ", err, s)
		}

		if i%100000 == 0 {
			log.Printf("loaded %v stop times", i)
		}
	}
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile | log.Ldate)
	db := common.DB

	// Dump stack trace on kill
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Println("got signal", sig)
			panic("stack trace here")
		}
	}()

	for _, url := range []string{
		"http://web.mta.info/developers/data/nyct/subway/google_transit.zip",
		"http://web.mta.info/developers/data/nyct/bus/google_transit_bronx.zip",
		"http://web.mta.info/developers/data/nyct/bus/google_transit_brooklyn.zip",
		"http://web.mta.info/developers/data/nyct/bus/google_transit_manhattan.zip",
		"http://web.mta.info/developers/data/nyct/bus/google_transit_queens.zip",
		"http://web.mta.info/developers/data/nyct/bus/google_transit_staten_island.zip",
	} {
		var stype string
		if strings.Contains(url, "subway") {
			stype = "subway"
		} else {
			stype = "bus"
		}

		// FIXME: do this in go, need to make it integrated with loader
		dir, err := ioutil.TempDir(os.Getenv("BUS_TMP_DIR"), "")
		if err != nil {
			panic(err)
		}
		cmd := exec.Command("/usr/bin/wget", url, "-O", path.Join(dir, "file.zip"))
		err = cmd.Run()
		if err != nil {
			panic(err)
		}

		cmd = exec.Command("/usr/bin/unzip", path.Join(dir, "file.zip"), "-d", dir)
		err = cmd.Run()
		if err != nil {
			panic(err)
		}

		func() {
			defer os.RemoveAll(dir)
			t1 := time.Now()
			doOne(dir, stype, db)
			t2 := time.Now()
			log.Printf("took %v for %v\n", t2.Sub(t1), dir)
		}()
	}

	log.Println("finished all boroughs")
}

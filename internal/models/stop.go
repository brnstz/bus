package models

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
	"github.com/jmoiron/sqlx"
)

// Stop is a single transit stop for a particular route. If a
// stop serves more than one route, there are multiple distinct
// entries for that stop.
type Stop struct {
	ID       string `json:"stop_id" db:"stop_id" upsert:"key"`
	RouteID  string `json:"route_id" db:"route_id" upsert:"key"`
	AgencyID string `json:"agency_id" db:"agency_id" upsert:"key"`
	Name     string `json:"stop_name" db:"stop_name"`

	DirectionID int    `json:"direction_id" db:"direction_id"`
	Headsign    string `json:"headsign" db:"headsign"`

	Lat float64 `json:"lat" db:"lat" upsert:"omit"`
	Lon float64 `json:"lon" db:"lon" upsert:"omit"`

	// Location is an "earth" field value that combines lat and lon into
	// a single field.
	Location interface{} `json:"-" db:"location" upsert_value:"ll_to_earth(:lat, :lon)"`

	// Dist, Scheduled, and Live fields that are only filled in
	// when returning a response from an API request. Do not directly
	// display them in JSON since
	Dist      float64      `json:"-" db:"dist" upsert:"omit"`
	Scheduled []*Departure `json:"-" db:"-" upsert:"omit"`
	Live      []*Departure `json:"-" db:"-" upsert:"omit"`
}

// Table implements the upsert.Upserter interface, returning the table
// where we save stops.
func (s *Stop) Table() string {
	return "stop"
}

// Save saves a stop to the database
func (s *Stop) Save() error {
	_, err := upsert.Upsert(etc.DBConn, s)
	return err
}

// appendLive calls either the bus time API or the subway datamine API
// to add live info to our stop info.
func (s *Stop) appendLive(now time.Time) {
	route, err := GetRoute(s.AgencyID, s.RouteID)
	if err != nil {
		log.Println("can't load route", err)
		return
	}

	if route.Type == Bus {
		departures, err := GetLiveBus(
			s.RouteID, strconv.Itoa(s.DirectionID),
			s.ID,
		)
		if err != nil {
			log.Println("can't append live schedules")
			return
		}

		sort.Sort(departures)

		for i := 0; i < len(departures) && i < maxDepartures; i++ {
			s.Live = append(s.Live, departures[i])
		}
	} else if route.Type == Subway {

		departures, err := GetLiveSubways(
			s.RouteID, strconv.Itoa(s.DirectionID),
			s.ID,
		)
		if err != nil {
			log.Println("can't append live subway sched", err)
			return
		}

		sort.Sort(departures)
		for i := 0; i < len(departures) && i < maxDepartures; i++ {
			s.Live = append(s.Live, departures[i])
		}

	}
}

// String returns a descriptive string for this stop.
func (s Stop) String() string {
	return fmt.Sprintf("{%v %v %v %v %v @ (%v,%v)}",
		s.ID, s.Name, s.RouteID, s.Headsign, s.DirectionID, s.Lat, s.Lon,
	)
}

// Key() returns the unique string for this stop, so we can identify
// unique stops in the loader.
func (s Stop) Key() string {
	return fmt.Sprintf("%v%v", s.ID, s.RouteID)
}

// setDepartures checks the database and any relevant APIs to set the scheduled
// and live departures for this stop
func (s *Stop) setDepartures(now time.Time, db sqlx.Ext) (err error) {

	allDepartures := Departures{}

	yesterday := baseTime(now.Add(-time.Hour * 12))
	today := baseTime(now)

	yesterdayName := strings.ToLower(yesterday.Format("Monday"))
	todayName := strings.ToLower(now.Format("Monday"))

	func() {
		if yesterdayName != todayName {
			var yesterdayID string
			// Looks for trips starting yesterday that arrive here
			// after midnight
			yesterdayID, err = getServiceIDByDay(
				db, s.RouteID, yesterdayName, yesterday,
			)
			if err == sql.ErrNoRows {
				err = nil
				log.Println("no rows, ok, moving on")
				return
			}
			if err != nil {
				log.Println("can't get yesterday id", err)
				return
			}

			nowSecs :=
				now.Hour()*3600 + now.Minute()*60 + now.Second() + midnightSecs

			departures, err := getDepartures(
				s.AgencyID, s.RouteID, s.ID, yesterdayID,
				nowSecs, yesterday)
			if err != nil {
				log.Println("can't get departures", err)
				return
			}

			log.Println("yesterday departures", departures)

			allDepartures = append(allDepartures, departures...)
		}
	}()

	func() {
		var todayID string
		todayID, err = getServiceIDByDay(db, s.RouteID, todayName, today)
		if err == sql.ErrNoRows {
			err = nil
			log.Println("no rows there", err)
			return
		}
		if err != nil {
			log.Println("can't get today id", err)
			return
		}

		nowSecs := now.Hour()*3600 + now.Minute()*60 + now.Second()

		departures, err := getDepartures(
			s.AgencyID, s.RouteID, s.ID, todayID,
			nowSecs, today)
		if err != nil {
			log.Println("can't get departures", err)
			return
		}

		allDepartures = append(allDepartures, departures...)

	}()

	sort.Sort(allDepartures)

	for i, d := range allDepartures {
		if i > maxDepartures {
			break
		}
		s.Scheduled = append(s.Scheduled, d)
	}

	// After reading scheduled times in the db, try to also append any
	// live info available
	s.appendLive(now)

	return
}

// GetStopsByQuery returns stops matching this StopQuery
func GetStopsByQuery(db sqlx.Ext, sq *StopQuery) (stops []Stop, err error) {
	now := time.Now()

	// Get rows matching the stop query
	rows, err := sqlx.NamedQuery(db, sq.Query(), sq)
	if err != nil {
		log.Println("can't get stops", err)
		return
	}

	for rows.Next() {
		var stop Stop

		err = rows.StructScan(&stop)
		if err != nil {
			log.Println("can't scan stop", err)
			return
		}

		err = stop.setDepartures(now, db)
		if err != nil {
			log.Println("can't set departures", err)
			return
		}

		stops = append(stops, stop)
	}

	return
}

func getServiceIDByDay(db sqlx.Ext, routeID, day string, now time.Time) (serviceID string, err error) {

	// Select the service_id that:
	//   * matches our routeID and day
	//   * has an end_date after now
	//   * has a start_date before now
	//   * if there's more than one, choose the one with the latest start_date

	row := db.QueryRowx(`
		SELECT service_id 
		FROM   service_route_day 
		WHERE  day = $1 AND
			   end_date >= $2 AND
			   start_date <= $3 AND 
			   route_id = $4
		ORDER BY start_date DESC
		LIMIT 1
	`,
		day, now, now, routeID,
	)

	err = row.Scan(&serviceID)
	if err != nil {
		log.Println("can't scan service id", err, day, now, routeID)
		return
	}

	return
}

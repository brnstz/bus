package models

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
	"github.com/jmoiron/sqlx"
)

var (
	maxStops = 20
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

	// StopSequence is the order in which this stop occurs in a typical
	// route trip, for comparisons with other stops matching the
	// same agency / route / stop / direction / headsign
	StopSequence int `json:"stop_sequence" db:"stop_sequence" upsert:"omit"`

	Dist      float64      `json:"dist" db:"-" upsert:"omit"`
	Scheduled []*Departure `json:"scheduled" db:"-" upsert:"omit"`
	Live      []*Departure `json:"live" db:"-" upsert:"omit"`
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
		if i > MaxDepartures {
			break
		}
		s.Scheduled = append(s.Scheduled, d)
	}

	return
}

// GetStop returns a single stop by its unique id
func GetStop(db sqlx.Ext, agencyID, routeID, stopID string, apppendInfo bool) (*Stop, error) {
	var s Stop
	now := time.Now()

	err := sqlx.Get(db, &s, `
		 SELECT stop.*, sst.stop_sequence,
				latitude(stop.location) AS lat,
				longitude(stop.location) AS lon

		 FROM stop
		 INNER JOIN route_trip ON route_trip.agency_id = stop.agency_id AND
		            			  route_trip.route_id  = stop.route_id
		 INNER JOIN scheduled_stop_time sst ON
								sst.agency_id = stop.agency_id     AND
								sst.route_id  = stop.route_id      AND
		            			sst.trip_id   = route_trip.trip_id AND
								sst.stop_id   = stop.stop_id  
		 WHERE stop.agency_id = $1 AND
			   stop.route_id  = $2 AND
			   stop.stop_id   = $3
		`, agencyID, routeID, stopID,
	)
	if err != nil {
		log.Println("can't get stop", err)
		return nil, err
	}

	err = s.setDepartures(now, db)
	if err != nil {
		log.Println("can't set departures", err)
		return nil, err
	}

	return &s, nil
}

// GetStopsByQuery returns stops matching this StopQuery
func GetStopsByQuery(db sqlx.Ext, sq StopQuery) (stops []*Stop, err error) {
	// distinct maps agency_id|route_id|direction_id to bool to ensure
	// we don't load duplicate routes
	distinct := map[string]bool{}

	// Get rows matching the stop query
	rows, err := sqlx.NamedQuery(db, sq.Query(), sq)
	if err != nil {
		log.Println("can't get stops", sq.Query(), err)
		return
	}

	defer rows.Close()

	count := 0
	for rows.Next() {
		var sqr stopQueryRow
		var stop *Stop

		if count >= maxStops {
			break
		}

		err = rows.StructScan(&sqr)
		if err != nil {
			log.Println("can't scan stop row", err)
			return
		}

		// skip duplicate rows
		if distinct[sqr.id()] {
			continue
		}
		distinct[sqr.id()] = true

		stop, err = GetStop(
			db, sqr.AgencyID, sqr.RouteID, sqr.StopID,
			sq.Departures,
		)
		if err != nil {
			log.Println("can't get stop", err)
			return
		}

		stop.Dist = sqr.Dist

		stops = append(stops, stop)
		count++
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

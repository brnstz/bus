package models

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
	"github.com/jmoiron/sqlx"
)

const (
	// minFirstDepartureSec is the minimum amount of time the first
	// departure must occur (2 hours)
	minFirstDepartureSec float64 = 60 * 60 * 2

	// departurePreWindow is how far in the past to look departures
	// that have already passed
	departurePreSec = 60 * 2
)

// Stop is a single transit stop for a particular route. If a
// stop serves more than one route, there are multiple distinct
// entries for that stop.
type Stop struct {
	StopID   string `json:"stop_id" db:"stop_id" upsert:"key"`
	RouteID  string `json:"route_id" db:"route_id" upsert:"key"`
	AgencyID string `json:"agency_id" db:"agency_id" upsert:"key"`
	Name     string `json:"stop_name" db:"stop_name"`

	UniqueID string `json:"unique_id" db:"-" upsert:"omit"`

	DirectionID int    `json:"direction_id" db:"direction_id"`
	Headsign    string `json:"headsign" db:"headsign"`

	Lat float64 `json:"lat" db:"lat" upsert:"omit"`
	Lon float64 `json:"lon" db:"lon" upsert:"omit"`

	// Location is PostGIS field value that combines lat and lon into a single
	// field.
	Location interface{} `json:"-" db:"location" upsert_value:"ST_SetSRID(ST_MakePoint(:lat, :lon),4326)"`

	Seq int `json:"seq" db:"stop_sequence" upsert:"omit"`

	Dist       float64      `json:"dist,omitempty" db:"-" upsert:"omit"`
	Departures []*Departure `json:"departures,omitempty" db:"-" upsert:"omit"`
	Vehicles   []Vehicle    `json:"vehicles,omitempty" db:"-" upsert:"omit"`
}

func (s *Stop) Initialize() error {
	s.UniqueID = s.AgencyID + "|" + s.RouteID + "|" + s.StopID

	return nil
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
		s.StopID, s.Name, s.RouteID, s.Headsign, s.DirectionID, s.Lat, s.Lon,
	)
}

// Key() returns the unique string for this stop, so we can identify
// unique stops in the loader.
func (s Stop) Key() string {
	return fmt.Sprintf("%v%v", s.StopID, s.RouteID)
}

// setDepartures checks the database and any relevant APIs to set the scheduled
// and live departures for this stop
func (s *Stop) setDepartures(now time.Time, db sqlx.Ext) (err error) {
	var yesterdayVehicles []Vehicle
	var todayVehicles []Vehicle

	if conf.API.LogTiming {
		t1 := time.Now()
		defer func() { log.Println(time.Now().Sub(t1)) }()
	}

	allDepartures := Departures{}

	yesterday := baseTime(now.Add(-time.Hour * 12))
	today := baseTime(now)

	yesterdayName := strings.ToLower(yesterday.Format("Monday"))
	todayName := strings.ToLower(now.Format("Monday"))

	func() {
		if yesterdayName != todayName {
			var yesterdayIDs []string
			// Looks for trips starting yesterday that arrive here
			// after midnight
			yesterdayIDs, err = getServiceIDsByDay(
				db, s.AgencyID, s.RouteID, yesterdayName, yesterday,
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

			nowSecs := now.Hour()*3600 + now.Minute()*60 + now.Second() + midnightSecs

			for _, yesterdayID := range yesterdayIDs {
				departures, err := getDepartures(
					s.AgencyID, s.RouteID, s.StopID, yesterdayID,
					nowSecs-departurePreSec, yesterday)
				if err != nil {
					log.Println("can't get departures", err)
					return
				}

				allDepartures = append(allDepartures, departures...)
			}

			yesterdayVehicles, err = getVehicles(s.AgencyID, s.RouteID, s.DirectionID, yesterdayIDs, nowSecs)
			if err != nil {
				log.Println("can't get vehicles", err)
				return
			}
		}
	}()

	func() {
		var todayIDs []string
		todayIDs, err = getServiceIDsByDay(db, s.AgencyID, s.RouteID, todayName, today)
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

		for _, todayID := range todayIDs {
			departures, err := getDepartures(
				s.AgencyID, s.RouteID, s.StopID, todayID,
				nowSecs-departurePreSec, today)
			if err != nil {
				log.Println("can't get departures", err)
				return
			}

			allDepartures = append(allDepartures, departures...)
		}

		todayVehicles, err = getVehicles(s.AgencyID, s.RouteID, s.DirectionID, todayIDs, nowSecs)
		if err != nil {
			log.Println("can't get vehicles", err)
			return
		}

	}()

	s.Vehicles = append(s.Vehicles, yesterdayVehicles...)
	s.Vehicles = append(s.Vehicles, todayVehicles...)

	// If there are no departures, we can return now
	if len(allDepartures) < 1 {
		return
	}

	// Sort departures by time
	sort.Sort(allDepartures)

	// Calculate the difference between now and the first
	// scheduled departure. Return if it's not soon enough.
	diff := allDepartures[0].Time.Sub(now)
	if diff.Seconds() > minFirstDepartureSec {
		return
	}

	// Add up to MaxDepartures to our scheduled list
	for i, d := range allDepartures {
		if i > MaxDepartures {
			break
		}
		s.Departures = append(s.Departures, d)
	}

	return
}

func GetStopsByTrip(db sqlx.Ext, t *Trip) (stops []*Stop, err error) {

	q := `
		SELECT stop.*, 
			ST_X(location) AS lat, 
			ST_Y(location) AS lon,
			sst.stop_sequence

		FROM stop
		INNER JOIN scheduled_stop_time sst 
			ON stop.agency_id = sst.agency_id AND
			   stop.route_id  = sst.route_id  AND
			   stop.stop_id   = sst.stop_id

		WHERE sst.agency_id     = $1 AND
			  sst.route_id      = $2 AND
	          sst.trip_id       = $3 AND
			  stop.direction_id = $4

		ORDER by sst.stop_sequence ASC
	`

	err = sqlx.Select(db, &stops, q,
		t.AgencyID, t.RouteID, t.TripID, t.DirectionID,
	)

	if err != nil {
		log.Println("can't get trips", err)
		return
	}

	return
}

// GetStop returns a single stop by its unique id
func GetStop(db sqlx.Ext, agencyID, routeID, stopID string, appendInfo bool) (*Stop, error) {
	var s Stop
	now := time.Now()

	if conf.API.LogTiming {
		t1 := time.Now()
		defer func() { log.Println(time.Now().Sub(t1)) }()
	}

	err := sqlx.Get(db, &s, `
		 SELECT stop.*, 
				ST_X(location) AS lat, 
				ST_Y(location) AS lon

		 FROM stop
		 WHERE stop.agency_id = $1 AND
			   stop.route_id  = $2 AND
			   stop.stop_id   = $3
		`, agencyID, routeID, stopID,
	)
	if err != nil {
		log.Println("can't get stop", err, agencyID, routeID, stopID)
		return nil, err
	}

	if appendInfo {
		err = s.setDepartures(now, db)
		if err != nil {
			log.Println("can't set departures", err)
			return nil, err
		}
	}

	return &s, nil
}

func getServiceIDsByDay(db sqlx.Ext, agencyID, routeID, day string, now time.Time) (serviceIDs []string, err error) {
	var normalIDs []string
	var addedIDs []string
	var removedIDs []string

	if conf.API.LogTiming {
		t1 := time.Now()
		defer func() { log.Println(time.Now().Sub(t1)) }()
	}

	removed := map[string]bool{}

	// FIXME: do we want to select all service IDs that are valid
	// in this time window or just the max start date (most recent one)
	// Experience with MTA data suggests we only want one.

	// Select the service_id that:
	//   * matches our agencyID, routeID, day
	//   * has an end_date after now
	//   * has a start_date before now
	//   * has the maximum start date of those

	q := `
		SELECT service_id 
		FROM   service_route_day 
		WHERE  day = $1 AND
			   end_date >= $2 AND
			   start_date <= $3 AND 
			   route_id = $4 AND
			   agency_id = $5
		ORDER BY start_date DESC
		LIMIT 1
	`

	err = sqlx.Select(db, &normalIDs, q, day, now, now, routeID, agencyID)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, routeID, agencyID)
		return
	}

	// Get services added / removed
	q = `
		SELECT service_id 
		FROM   service_route_exception
		WHERE  exception_date = $1 AND
			   route_id = $2 AND
			   agency_id = $3 AND
			   exception_type = $4
	`

	// Added
	err = sqlx.Select(db, &addedIDs, q, now, routeID, agencyID, ServiceAdded)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, routeID, agencyID, ServiceAdded)
		return
	}

	// Removed
	err = sqlx.Select(db, &removedIDs, q, now, routeID, agencyID, ServiceRemoved)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, routeID, agencyID, ServiceRemoved)
		return
	}

	for _, v := range removedIDs {
		removed[v] = true
	}

	for _, v := range normalIDs {
		if !removed[v] {
			serviceIDs = append(serviceIDs, v)
		}
	}

	for _, v := range addedIDs {
		if !removed[v] {
			serviceIDs = append(serviceIDs, v)
		}
	}

	return
}

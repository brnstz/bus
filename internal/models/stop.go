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

const (
	maxStops = 3
)

// Stop is a single transit stop for a particular route. If a
// stop serves more than one route, there are multiple distinct
// entries for that stop.
type Stop struct {
	ID      string `json:"stop_id" db:"stop_id" upsert:"key"`
	RouteID string `json:"route_id" db:"route_id" upsert:"key"`
	Name    string `json:"stop_name" db:"stop_name"`

	DirectionID int    `json:"direction_id" db:"direction_id"`
	Headsign    string `json:"headsign" db:"headsign"`

	Lat float64 `json:"lat" db:"lat" upsert:"omit"`
	Lon float64 `json:"lon" db:"lon" upsert:"omit"`

	// Location is an "earth" field value that combines lat and lon into
	// a single field.
	Location interface{} `json:"-" db:"location" upsert_value:"ll_to_earth(:lat, :lon)"`

	// Dist, Scheduled, and Live and columns that are only filled in
	// when returning a response from an API request.

	Dist      float64      `json:"dist" db:"dist" upsert:"omit"`
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

// AppendLive calls either the bus time API or the subway datamine API
// to add live info to our stop info.
func (s *Stop) AppendLive(now time.Time) {
	route, err := GetRoute(s.RouteID)
	if err != nil {
		log.Println("can't load route", err)
		return
	}

	if route.Type == Bus {
		calls, err := GetCallsByRouteStop(
			s.RouteID, strconv.Itoa(s.DirectionID),
			s.ID,
		)
		if err != nil {
			log.Println("can't append live schedules")
			return
		}

		sort.Sort(calls)
		for i := 0; i < len(calls) && i < maxStops; i++ {
			s.Live = append(s.Live, &Departure{
				Desc: calls[i].Extensions.Distances.PresentableDistance,
			})
		}
	} else if route.Type == Subway {

		times, err := GetLiveSubways(
			s.RouteID, strconv.Itoa(s.DirectionID),
			s.ID,
		)
		if err != nil {
			log.Println("can't append live subway sched", err)
			return
		}

		sort.Sort(times)
		for i := 0; i < len(times) && i < maxStops; i++ {
			s.Live = append(s.Live, &Departure{
				Time: times[i],
			})
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

	ydaysecs := []int64{}
	todaysecs := []int64{}

	allTimes := timeSlice{}

	yesterday := now.Add(-time.Hour * 12)
	yesterdayName := strings.ToLower(yesterday.Format("Monday"))
	todayName := strings.ToLower(now.Format("Monday"))

	func() {
		if yesterdayName != todayName {
			var yesterdayID string
			// Looks for trips starting yesterday that arrive here
			// after midnight
			yesterdayID, err = getServiceIDByDay(
				db, s.RouteID, yesterdayName, &now,
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

			qYesterday := `
			SELECT scheduled_stop_time.departure_sec
			FROM   scheduled_stop_time
			WHERE  route_id   = $1 AND
			       stop_id    = $2 AND
				   service_id = $3 AND
				   departure_sec >= 86400 AND
				   departure_sec > $4
			ORDER BY departure_sec LIMIT $5
		`
			nowSecs :=
				now.Hour()*3600 + now.Minute()*60 + now.Second() + 86400

			err = sqlx.Select(db, &ydaysecs,
				qYesterday, s.RouteID, s.ID,
				yesterdayID, nowSecs, maxStops)

			if err != nil {
				log.Println("can't scan yesterday values", err)
				return
			}

			yesterday = yesterday.Add(
				-time.Hour * time.Duration(yesterday.Hour()))
			yesterday = yesterday.Add(
				-time.Minute * time.Duration(yesterday.Minute()))
			yesterday = yesterday.Add(
				-time.Second * time.Duration(yesterday.Second()))
			yesterday = yesterday.Add(
				-time.Nanosecond * time.Duration(yesterday.Nanosecond()))

			for _, ydaysec := range ydaysecs {
				thisTime := yesterday.Add(time.Second * time.Duration(ydaysec))
				allTimes = append(allTimes, thisTime)
			}
		}
	}()

	func() {
		var todayID string
		todayID, err = getServiceIDByDay(db, s.RouteID, todayName, &now)
		if err == sql.ErrNoRows {
			err = nil
			log.Println("no rows there", err)
			return
		}
		if err != nil {
			log.Println("can't get today id", err)
			return
		}

		qToday := `
		SELECT scheduled_stop_time.departure_sec
		FROM   scheduled_stop_time
		WHERE  route_id   = $1 AND
			   stop_id    = $2 AND
			   service_id = $3 AND
			   departure_sec > $4
		ORDER BY departure_sec LIMIT $5
	`

		nowSecs := now.Hour()*3600 + now.Minute()*60 + now.Second()
		err = sqlx.Select(db, &todaysecs, qToday, s.RouteID, s.ID,
			todayID, nowSecs, maxStops)

		today := now
		today = today.Add(
			-time.Hour * time.Duration(today.Hour()))
		today = today.Add(
			-time.Minute * time.Duration(today.Minute()))
		today = today.Add(
			-time.Second * time.Duration(today.Second()))
		today = today.Add(
			-time.Nanosecond * time.Duration(today.Nanosecond()))

		for _, todaysec := range todaysecs {
			thisTime := today.Add(time.Second * time.Duration(todaysec))
			allTimes = append(allTimes, thisTime)
		}

		if err != nil {
			log.Println("can't scan today values", err)
			return
		}
	}()

	sort.Sort(allTimes)

	for i, thisTime := range allTimes {
		if i > maxStops {
			break
		}
		s.Scheduled = append(
			s.Scheduled, &Departure{Time: thisTime},
		)
	}

	// After reading scheduled times in the db, try to also append any
	// live info available
	s.AppendLive(now)

	return
}

// GetStopsByLoc returns a list of stops within meters of this lat and lon
func GetStopsByLoc(db sqlx.Ext, lat, lon, meters float64, filter string) (stops []*Stop, err error) {

	stops = []*Stop{}
	params := []interface{}{lat, lon, lat, lon, meters}

	q := `
		SELECT * FROM (
			SELECT
				DISTINCT ON (stop.route_id, direction_id)
				stop_id,
				stop_name,
				direction_id,
				headsign,
				stop.route_id,
				latitude(location) AS lat,
				longitude(location) AS lon,
				earth_distance(location, ll_to_earth($1, $2)) AS dist
			FROM stop INNER JOIN
			     route ON stop.route_id = route.route_id
			WHERE earth_box(ll_to_earth($3, $4), $5) @> location
	`

	if len(filter) > 0 {
		q = q + ` AND route.route_type = $6 `
		params = append(params, routeTypeInt[filter])
	}

	q = q + ` 
			ORDER BY stop.route_id, direction_id, dist
		) unique_routes

		ORDER BY dist ASC
	`

	err = sqlx.Select(db, &stops, q, params...)
	if err != nil {
		log.Println("can't get stop", err)
		return
	}

	now := time.Now()
	for _, stop := range stops {
		// FIXME: handle error here?
		stop.setDepartures(now, db)
	}

	return stops, err
}

// FIXME: is this correct?
func getServiceIDByDay(db sqlx.Ext, routeId, day string, now *time.Time) (serviceId string, err error) {
	row := db.QueryRowx(`
		SELECT service_id, route_id, max(start_date)
		FROM   service_route_day
		WHERE  day         = $1 AND
		       end_date    > $2 AND
			   route_id    = $3
		GROUP BY service_id, route_id
		LIMIT 1
	`, day, now, routeId,
	)

	var dummy1 string
	var dummy2 time.Time

	err = row.Scan(&serviceId, &dummy1, &dummy2)
	if err != nil {
		log.Println("can't scan service id", err, day, now, routeId)
		return
	}

	return
}

type timeSlice []time.Time

func (p timeSlice) Len() int {
	return len(p)
}

func (p timeSlice) Less(i, j int) bool {
	return p[i].Before(p[j])
}

func (p timeSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

package models

import (
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

type sortableStops []*Stop

func (ss sortableStops) Len() int {
	return len(ss)
}

func (ss sortableStops) Less(i, j int) bool {
	s1 := ss[i]
	s2 := ss[j]

	return s1.Dist < s2.Dist
}

func (ss sortableStops) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}

func GetStopsByHereQuery(db sqlx.Ext, hq HereQuery) (stops []*Stop, err error) {
	ss := sortableStops{}

	// mapping of stop.UniqueID to stop
	sm := map[string]*Stop{}

	// mapping of route.UniqueID + DirectionID to route
	rm := map[string]*Route{}

	// overall function timing
	if conf.API.LogTiming {
		t1 := time.Now()
		defer func() { log.Println(time.Now().Sub(t1)) }()
	}

	now := time.Now()

	//yesterday := etc.BaseTime(now.Add(-time.Hour * 12))
	today := etc.BaseTime(now)

	//yesterdayName := strings.ToLower(yesterday.Format("Monday"))
	todayName := strings.ToLower(now.Format("Monday"))

	// FIXME: hard coded, we need a region to agency mapping
	agencyID := "MTA NYCT"

	todayIDs, err := getNewServiceIDs(db, agencyID, todayName, today)
	if err != nil {
		log.Println("can't get today IDs", err)
		return
	}

	/* FIXME
	yesterdayIDs, err := getNewServiceIDs(db, agencyID, yesterdayName, yesterday)
	if err != nil {
		log.Println("can't get yesterday IDs", err)
		return
	}
	*/

	hq.ServiceIDs = todayIDs
	//hq.YesterdayServiceIDs = yesterdayIDs

	/*
		hq.YesterdayDepartureMin = now.Hour()*3600 + now.Minute()*60 + now.Second() + midnightSecs
		hq.YesterdayDepartureMax = hq.YesterdayDepartureMin + 60*60*3
	*/

	hq.DepartureMin = now.Hour()*3600 + now.Minute()*60 + now.Second()
	hq.DepartureMax = hq.DepartureMin + 60*60*3

	err = hq.Initialize()
	if err != nil {
		log.Println("can't initialize hq", err)
		return
	}

	t3 := time.Now()
	rows, err := sqlx.NamedQuery(db, hq.Query, hq)
	if err != nil {
		log.Println("can't get stops", err)
		log.Printf("%s %+v", hq.Query, hq)
		return
	}
	if conf.API.LogTiming {
		log.Println(time.Now().Sub(t3))
	}

	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
		here := HereResult{DepartureBase: today}

		err = rows.StructScan(&here)
		if err != nil {
			log.Println("can't scan row", err)
			continue
		}

		err = here.Initialize()
		if err != nil {
			log.Println("can't initialize here", err)
			continue
		}

		routeDir := fmt.Sprintf("%v|%v", here.Route.UniqueID, here.Stop.DirectionID)

		oldStop, stopExists := sm[here.Stop.UniqueID]
		//oldRoute, routeExists := rm[routeDir]
		_, routeExists := rm[routeDir]

		// Ignore when the route / direction already exists, but stop is not
		// the same
		if routeExists && !stopExists {
			continue
		}

		// Ignore if it's our stop but we already have too many departures
		if stopExists && len(oldStop.Departures) >= MaxDepartures {
			continue
		}

		if !stopExists {
			sm[here.Stop.UniqueID] = here.Stop
		}
		if !routeExists {
			rm[routeDir] = here.Route
		}

		stop := sm[here.Stop.UniqueID]
		//route := rm[routeDir]

		stop.Departures = append(stop.Departures, here.Departure)
	}

	// Add all stops to sortableStops list
	for _, s := range sm {
		ss = append(ss, s)
	}

	// sort stops by distance
	sort.Sort(ss)

	stops = []*Stop(ss)

	for _, r := range rm {
		log.Println("what is the route?", r)
	}

	return
}

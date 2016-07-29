package models

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/jmoiron/sqlx"
)

const (
	maxStops = 20
)

type HereResult struct {
	AgencyID  string `db:"agency_id"`
	RouteID   string `db:"route_id"`
	StopID    string `db:"stop_id"`
	ServiceID string `db:"service_id"`

	TripIDs       string `db:"trip_ids"`
	ArrivalSecs   string `db:"arrival_secs"`
	DepartureSecs string `db:"departure_secs"`
	StopSequences string `db:"stop_sequences"`
	CompassDirs   string `db:"compass_dirs"`

	TripHeadsign string `db:"trip_headsign"`

	StopName     string  `db:"stop_name"`
	StopHeadsign string  `db:"stop_headsign"`
	DirectionID  int     `db:"direction_id"`
	Lat          float64 `db:"lat"`
	Lon          float64 `db:"lon"`
	Dist         float64 `db:"dist"`

	RouteType      int    `db:"route_type"`
	RouteColor     string `db:"route_color"`
	RouteTextColor string `db:"route_text_color"`

	HQ *HereQuery

	Stop  *Stop
	Route *Route

	Departures []*Departure
}

func (h *HereResult) createStop() (stop *Stop, err error) {
	stop = &Stop{
		StopID:      h.StopID,
		RouteID:     h.RouteID,
		AgencyID:    h.AgencyID,
		Name:        h.StopName,
		DirectionID: h.DirectionID,
		Headsign:    h.StopHeadsign,
		Lat:         h.Lat,
		Lon:         h.Lon,
		Dist:        h.Dist,

		RouteType:      h.RouteType,
		RouteColor:     h.RouteColor,
		RouteTextColor: h.RouteTextColor,

		// FIXME: is seq even needed?
		//Seq: h.StopSequence,
	}
	err = stop.Initialize()
	if err != nil {
		log.Println("can't init stop", err)
		return
	}

	return
}

func (h *HereResult) createRoute() (route *Route, err error) {
	route = &Route{
		RouteID:   h.RouteID,
		AgencyID:  h.AgencyID,
		Type:      h.RouteType,
		Color:     h.RouteColor,
		TextColor: h.RouteTextColor,
	}

	err = route.Initialize()
	if err != nil {
		log.Println("can't init route", err)
		return
	}

	return
}

func (h *HereResult) Initialize() error {
	var err error

	h.Stop, err = h.createStop()
	if err != nil {
		log.Println("can't init here stop", err)
		return err
	}

	h.Route, err = h.createRoute()
	if err != nil {
		log.Println("can't init here route", err)
		return err
	}

	h.Departures, err = h.createDepartures()
	if err != nil {
		log.Println("can't init departure", err)
		return err
	}

	return nil
}

func (h *HereResult) createDepartures() (departures []*Departure, err error) {
	var (
		departureSec  int
		compassDir    float64
		tripID        string
		departureBase time.Time
	)

	departureSecs := strings.Split(h.DepartureSecs, ",")
	tripIDs := strings.Split(h.TripIDs, ",")
	compassDirs := strings.Split(h.CompassDirs, ",")

	if len(departureSecs) < 1 {
		err = fmt.Errorf("invalid departureSecs: %v", h.DepartureSecs)
		return
	}
	if departureSecs[0] == "" {
		err = fmt.Errorf("empty departure secs")
		return
	}
	if len(departureSecs) != len(tripIDs) {
		err = fmt.Errorf("mismatch between departureSecs length (%v) and tripIDs length (%v)", len(departureSecs), len(tripIDs))
		return
	}

	if len(departureSecs) != len(compassDirs) {
		log.Println(h)
		log.Println(h.CompassDirs)
		log.Println(compassDirs)
		err = fmt.Errorf("mismatch between departureSecs length (%v) and compassDirs length (%v)", len(departureSecs), len(compassDirs))
		return
	}

	for i := range departureSecs {
		departureSec, err = strconv.Atoi(strings.TrimSpace(departureSecs[i]))
		if err != nil {
			log.Println("can't parse departure sec", err)
			return
		}
		tripID = strings.TrimSpace(tripIDs[i])
		compassDir, err = strconv.ParseFloat(compassDirs[i], 64)
		if err != nil {
			log.Println("can't parse compass dir", err)
			return
		}

		// We have up to three non-overlapping ranges of departure sec,
		// that could be yesterday, today or tomorrow. We're able to do this
		// because the range is only 3 hours.

		if departureSec >= h.HQ.YesterdayDepartureMin &&
			departureSec <= h.HQ.YesterdayDepartureMax &&
			h.HQ.YesterdayServiceIDMap[h.ServiceID] {
			departureBase = h.HQ.YesterdayDepartureBase

		} else if departureSec >= h.HQ.TodayDepartureMin &&
			departureSec <= h.HQ.TodayDepartureMax &&
			h.HQ.TodayServiceIDMap[h.ServiceID] {
			departureBase = h.HQ.TodayDepartureBase

		} else if departureSec >= h.HQ.TomorrowDepartureMin &&
			departureSec <= h.HQ.TomorrowDepartureMax &&
			h.HQ.TomorrowServiceIDMap[h.ServiceID] {
			departureBase = h.HQ.TomorrowDepartureBase

		} else {
			// If it's not in our range, then we ignore it
			continue
		}

		departure := &Departure{
			DepartureSec: departureSec,
			TripID:       tripID,
			ServiceID:    h.ServiceID,
			baseTime:     departureBase,
			CompassDir:   compassDir,
		}

		err = departure.Initialize()
		if err != nil {
			log.Println("can't init departure", err)
			return
		}

		departures = append(departures, departure)
	}

	return
}

func GetHereResults(db sqlx.Ext, hq *HereQuery) (stops []*Stop, stopRoutes map[string]*Route, err error) {

	// mapping of stop.UniqueID to route
	stopRoutes = map[string]*Route{}

	// mapping of stop.UniqueID to stop
	sm := map[string]*Stop{}

	// mapping of route.UniqueID + DirectionID to route
	rm := map[string]*Route{}

	// overall function timing
	if conf.API.LogTiming {
		t1 := time.Now()
		defer func() { log.Println(time.Now().Sub(t1)) }()
	}

	t3 := time.Now()
	rows, err := sqlx.NamedQuery(db, hq.Query, hq)
	if err != nil {
		log.Println("can't get stops", err)
		log.Printf("%s %+v", hq.Query, hq)
		return
	}
	queryDur := time.Now().Sub(t3)
	if conf.API.LogTiming && queryDur > time.Duration(1)*time.Second {
		log.Printf("long here query (%v): %+v", queryDur, hq)
	}

	defer rows.Close()

	count := 0
	for rows.Next() {
		here := HereResult{HQ: hq}

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

		for _, departure := range here.Departures {

			routeDir := fmt.Sprintf("%v|%v", here.Route.UniqueID, here.Stop.DirectionID)

			_, stopExists := sm[here.Stop.UniqueID]
			_, routeExists := rm[routeDir]

			// Ignore when the route / direction already exists, but stop is not
			// the same
			if routeExists && !stopExists {
				continue
			}

			// If we didn't have stop or route, put them in our map
			if !stopExists {
				sm[here.Stop.UniqueID] = here.Stop
			}
			if !routeExists {
				rm[routeDir] = here.Route
			}

			// Get the stop and append the current departure
			stop := sm[here.Stop.UniqueID]
			stop.Departures = append(stop.Departures, departure)
			stopRoutes[here.Stop.UniqueID] = here.Route

			count++
		}

	}

	// Add all stops to sortableStops list
	for _, s := range sm {
		stops = append(stops, s)
	}

	ss := newSortableStops(stops)

	// Our goal is to sort
	// - Trains before buses
	// - Then by distance
	// - Grouped by route_id
	// - With consistent direction id 0/1 ordering
	ss.by = byDir
	sort.Sort(ss)

	ss.by = byRoute
	sort.Stable(ss)

	ss.by = byDist
	sort.Stable(ss)

	ss.by = byType
	sort.Stable(ss)

	// Assign stops to our return value
	if len(ss.stops) > maxStops {
		stops = []*Stop(ss.stops[0:maxStops])
	} else {
		stops = []*Stop(ss.stops)
	}

	// For each stop, sort its departures and limit to max number of departures
	for _, s := range stops {
		d := SortableDepartures(s.Departures)
		sort.Sort(d)

		if len(d) > MaxDepartures {
			s.Departures = []*Departure(d[0:MaxDepartures])
		} else {
			s.Departures = []*Departure(d)
		}
	}

	return
}

package models

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/jmoiron/sqlx"
)

const (
	maxStops              = 20
	minFirstDepartureTime = time.Duration(2) * time.Hour
)

type HereResult struct {
	// shared fields
	AgencyID  string `db:"agency_id"`
	RouteID   string `db:"route_id"`
	StopID    string `db:"stop_id"`
	TripID    string `db:"trip_id"`
	ServiceID string `db:"service_id"`

	// Departure
	ArrivalSec   int `db:"arrival_sec"`
	DepartureSec int `db:"departure_sec"`

	// Trip
	StopSequence int    `db:"stop_sequence"`
	TripHeadsign string `db:"trip_headsign"`

	// Stop
	StopName     string  `db:"stop_name"`
	StopHeadsign string  `db:"stop_headsign"`
	DirectionID  int     `db:"direction_id"`
	Lat          float64 `db:"lat"`
	Lon          float64 `db:"lon"`
	Dist         float64 `db:"dist"`

	// Route
	RouteType      int    `db:"route_type"`
	RouteColor     string `db:"route_color"`
	RouteTextColor string `db:"route_text_color"`

	DepartureBase time.Time

	Stop      *Stop
	Route     *Route
	Departure *Departure
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
		Seq: h.StopSequence,
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

func (h *HereResult) createDeparture() (departure *Departure, err error) {
	departure = &Departure{
		DepartureSec: h.DepartureSec,
		TripID:       h.TripID,
		ServiceID:    h.ServiceID,
		baseTime:     h.DepartureBase,
	}

	err = departure.Initialize()
	if err != nil {
		log.Println("can't init departure", err)
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

	h.Departure, err = h.createDeparture()
	if err != nil {
		log.Println("can't init here departure", err)
		return err
	}

	return nil

}

func GetHereResults(db sqlx.Ext, hq *HereQuery) (stops []*Stop, stopRoutes map[string]*Route, err error) {
	ss := sortableStops{}

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
		here := HereResult{}

		err = rows.StructScan(&here)
		if err != nil {
			log.Println("can't scan row", err)
			continue
		}

		if here.DepartureSec >= hq.YesterdayDepartureMin &&
			here.DepartureSec <= hq.YesterdayDepartureMax {

			here.DepartureBase = hq.YesterdayDepartureBase

		} else if here.DepartureSec >= hq.TodayDepartureMin &&
			here.DepartureSec <= hq.TodayDepartureMax {

			here.DepartureBase = hq.TodayDepartureBase
		} else if here.DepartureSec >= hq.TomorrowDepartureMin &&
			here.DepartureSec <= hq.TomorrowDepartureMax {

			here.DepartureBase = hq.TomorrowDepartureBase
		}

		err = here.Initialize()
		if err != nil {
			log.Println("can't initialize here", err)
			continue
		}

		routeDir := fmt.Sprintf("%v|%v", here.Route.UniqueID, here.Stop.DirectionID)

		oldStop, stopExists := sm[here.Stop.UniqueID]
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

		// If we didn't have stop or route, put them in our map
		if !stopExists {
			sm[here.Stop.UniqueID] = here.Stop
		}
		if !routeExists {
			rm[routeDir] = here.Route
		}

		// Get the stop and append the current departure
		stop := sm[here.Stop.UniqueID]
		stop.Departures = append(stop.Departures, here.Departure)
		stopRoutes[here.Stop.UniqueID] = here.Route

		count++

	}

	// Add all stops to sortableStops list
	for _, s := range sm {
		ss.stops = append(ss.stops, s)
	}

	// sort stops by distance first
	ss.by = byDist
	sort.Stable(ss)

	// then sort by type to put subways first
	ss.by = byType
	sort.Stable(ss)

	// Assign stops to our return value
	if len(ss.stops) > maxStops {
		stops = []*Stop(ss.stops[0:maxStops])
	} else {
		stops = []*Stop(ss.stops)
	}

	return
}

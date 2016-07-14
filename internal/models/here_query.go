package models

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"
)

var (
	defaultMaxResults = 1000

	// minFirstDepartureSec is the minimum amount of time the first
	// departure must occur (2 hours)
	minFirstDepartureSec float64 = 60 * 60 * 2

	// departurePreWindow is how far in the past to look departures
	// that have already passed
	departurePreSec = 60 * 2
)

type HereResult struct {
	// shared fields
	AgencyID  string `db:"agency_id"`
	RouteID   string `db:"route_id"`
	StopID    string `db:"stop_id"`
	TripID    string `db:"trip_id"`
	ServiceID string `db:"service_id"`

	// Departure
	ArrivalSec  int `db:"arrival_sec"`
	DepatureSec int `db:"departure_sec"`

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
		DepartureSec: h.DepatureSec,
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

const (
	hereQuery = `
		SELECT
			agency_id,
			route_id,
			stop_id,
			service_id,
			trip_id,
			arrival_sec,
			departure_sec,
			stop_sequence,

			stop_name,
			direction_id,
			stop_headsign,
			ST_X(location) AS lat,
			ST_Y(location) AS lon,

			route_type,
			route_color,
			route_text_color,

			trip_headsign,

			ST_DISTANCE(ST_GEOMFROMTEXT(:point_string, 4326), location) AS dist

		FROM here

		WHERE
			ST_CONTAINS(ST_SETSRID(ST_MAKEPOLYGON(:line_string), 4326), location) AND

			(
				(   
					service_id IN (%s) AND
					departure_sec > :departure_min AND
					departure_sec < :departure_max
				)
			)
		ORDER BY dist ASC, departure_sec ASC
		LIMIT :limit
	`
)

type HereQuery struct {
	// The southwest and northeast bounding points of the box we are
	// searching
	SWLat float64 `db:"sw_lat"`
	SWLon float64 `db:"sw_lon"`
	NELat float64 `db:"ne_lat"`
	NELon float64 `db:"ne_lon"`

	// The midpoint of our search box
	MidLat float64 `db:"mid_lat"`
	MidLon float64 `db:"mid_lon"`

	LineString  string `db:"line_string"`
	PointString string `db:"point_string"`

	ServiceIDs []string

	DepartureMin int `db:"departure_min"`
	DepartureMax int `db:"departure_max"`

	Limit int `db:"limit"`

	Query string
}

// createIDs turns a slice of service IDs into a single string suitable
// for substitution into an IN clause.
func createIDs(ids []string) string {
	// If there are no ids, we want a single blank value
	if len(ids) < 1 {
		return `''`
	}

	for i, _ := range ids {
		ids[i] = escape(ids[i])
	}

	return strings.Join(ids, ",")
}

// escape ensures any single quotes inside of serviceID are escaped / quoted
// before creating an ad-hoc string for the IN query
func escape(serviceID string) string {
	var b bytes.Buffer

	b.WriteRune('\u0027')

	for _, char := range serviceID {
		switch char {
		case '\u0027':
			b.WriteRune('\u0027')
			b.WriteRune('\u0027')
		default:
			b.WriteRune(char)
		}
	}

	b.WriteRune('\u0027')

	return b.String()
}

func (hq *HereQuery) Initialize() error {

	hq.LineString = fmt.Sprintf(
		`LINESTRING(%f %f, %f %f, %f %f, %f %f, %f %f)`,
		hq.SWLat, hq.SWLon,
		hq.SWLat, hq.NELon,
		hq.NELat, hq.NELon,
		hq.NELat, hq.SWLon,
		hq.SWLat, hq.SWLon,
	)

	hq.PointString = fmt.Sprintf(
		`POINT(%f %f)`,
		hq.MidLat, hq.MidLon,
	)

	hq.Query = fmt.Sprintf(hereQuery,
		createIDs(hq.ServiceIDs),
	)

	if hq.Limit < 1 {
		hq.Limit = defaultMaxResults
	}

	return nil
}

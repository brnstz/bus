package models

import (
	"bytes"
	"fmt"
	"strings"
)

var (
	defaultMaxResults = 1000
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

			ST_DISTANCE(ST_GEOMFROMTEXT(:point_string, :srid), location) AS dist

		FROM here

		WHERE
			ST_CONTAINS(ST_SETSRID(ST_MAKEPOLYGON(:line_string), :srid), location) AND

			(
				(   
					service_id IN (%s) AND
					departure_sec > :today_departure_min AND
					departure_sec < :today_departure_max
				) OR

				(   service_id IN (%s) AND
					departure_sec > :yesterday_departure_min AND
					departure_sec < :yesterday_departure_max
				)
			)
		ORDER BY dist ASC, departure_sec ASC
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

	TodayServiceIDs     []string
	YesterdayServiceIDs []string

	TodayDepartureMin int `db:"today_departure_min"`
	TodayDepartureMax int `db:"today_departure_max"`

	YesterdayDepartureMin int `db:"yesterday_departure_min"`
	YesterdayDepartureMax int `db:"yesterday_departure_max"`

	SRID int `db:"srid"`

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

	hq.SRID = 4326

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
		createIDs(hq.TodayServiceIDs),
		createIDs(hq.YesterdayServiceIDs),
	)

	return nil
}

package models

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

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

	DepartureBase time.Time

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

func NewHereQuery(lat, lon, swlat, swlon, nelat, nelon float64, serviceIDs []string, minSec int, departureBase time.Time) (hq *HereQuery, err error) {
	hq = &HereQuery{
		MidLat:        lat,
		MidLon:        lon,
		SWLat:         swlat,
		SWLon:         swlon,
		NELat:         nelat,
		NELon:         nelon,
		ServiceIDs:    serviceIDs,
		Limit:         2000,
		DepartureMin:  minSec,
		DepartureMax:  minSec + 60*60*6,
		DepartureBase: departureBase,
	}

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

	return
}

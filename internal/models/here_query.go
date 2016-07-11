package models

import (
	"bytes"
	"fmt"
	"strings"
)

var (
	defaultMaxResults = 1000
)

const (
	hereQuery = `
		SELECT
			here.*, ST_DISTANCE(ST_GEOMFROMTEXT(:point_string, :srid), location) AS dist

		FROM here

		WHERE
			ST_CONTAINS(ST_SETSRID(ST_MAKEPOLYGON(:line_string), :srid), GEOMETRY(location)) AND

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
		ORDER BY dist ASC
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

	LineString string `db:"line_string"`

	TodayServiceIDs     []string
	YesterdayServiceIDs []string

	TodayDepartureMin int `db:"today_departure_min"`
	TodayDepartureMax int `db:"today_departure_max"`

	YesterdayDepartureMin int `db:"yesterday_departure_min"`
	YesterdayDepartureMax int `db:"yesterday_departure_max"`

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

	hq.Query = fmt.Sprintf(hereQuery,
		createIDs(hq.TodayServiceIDs),
		createIDs(hq.YesterdayServiceIDs),
	)

	return nil
}

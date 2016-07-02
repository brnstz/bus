package models

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	defaultMaxStops = 20
)

const (
	sqBegin = `
		SELECT stop.agency_id, stop.route_id, stop.stop_id,
		       stop.direction_id,
			   ST_Distance(location, ST_SetSRID(ST_MakePoint(:mid_lat, :mid_lon),4326)) AS dist
		FROM stop
		INNER JOIN route ON stop.route_id = route.route_id
	`

	sqLocationFilter = `
		ST_Contains(ST_SetSRID(ST_MakePolygon(:line_string), 4326), geometry(location))
	`

	sqRouteFilter = ` 
		route.route_type = :route_type
	`

	sqRouteIDFilter = `
		route.route_id = :route_id
	`

	sqAgencyIDFilter = `
		route.agency_id = :agency_id
	`

	sqEnd = ` 
		ORDER BY dist ASC LIMIT 100
	`
)

type stopQueryRow struct {
	AgencyID    string  `db:"agency_id"`
	RouteID     string  `db:"route_id"`
	StopID      string  `db:"stop_id"`
	DirectionID int     `db:"direction_id"`
	Dist        float64 `db:"dist"`
}

func (sqr *stopQueryRow) id() string {
	return sqr.AgencyID + "|" + sqr.RouteID + "|" + strconv.Itoa(sqr.DirectionID)
}

type StopQuery struct {
	// The southwest and northeast bounding points of the box we are
	// searching
	SWLat float64 `db:"sw_lat"`
	SWLon float64 `db:"sw_lon"`
	NELat float64 `db:"ne_lat"`
	NELon float64 `db:"ne_lon"`

	// The midpoint of our search box
	MidLat float64 `db:"mid_lat"`
	MidLon float64 `db:"mid_lon"`

	// filter on this specific RouteType if specified
	RouteType string `db:"-"`

	RouteTypeInt int `db:"route_type"`

	// append departures to returned stops
	Departures bool

	// only return distinct entries (agency_id|route_id|direction_id)
	Distinct bool

	// only return this many stops, defaults to  defaultMaxStops
	MaxStops int

	RouteID  string `db:"route_id"`
	AgencyID string `db:"agency_id"`

	LineString  string `db:"line_string"`
	routeFilter bool
}

// initialize checks that a StopQuery has sane values and any private
// variables are initialized
func (sq *StopQuery) Initialize() error {

	if len(sq.RouteType) > 0 {
		var ok bool
		sq.routeFilter = true
		sq.RouteTypeInt, ok = routeTypeInt[sq.RouteType]
		if !ok {
			return ErrInvalidRouteType
		}
	}

	if sq.MaxStops < 1 {
		sq.MaxStops = defaultMaxStops
	}

	sq.LineString = fmt.Sprintf(
		`LINESTRING(%f %f, %f %f, %f %f, %f %f, %f %f)`,
		sq.SWLat, sq.SWLon,
		sq.SWLat, sq.NELon,
		sq.NELat, sq.NELon,
		sq.NELat, sq.SWLon,
		sq.SWLat, sq.SWLon,
	)

	return nil
}

// Query returns the SQL for this StopQuery
func (sq *StopQuery) Query() string {
	var where []string
	var whereClause string

	if sq.routeFilter {
		where = append(where, sqRouteFilter)
	}

	if len(sq.RouteID) > 0 {
		where = append(where, sqRouteIDFilter)
	}

	if len(sq.AgencyID) > 0 {
		where = append(where, sqAgencyIDFilter)
	}

	if !(sq.SWLat == 0.0 && sq.SWLon == 0.0 && sq.NELat == 0.0 && sq.NELon == 0.0) {

		where = append(where, sqLocationFilter)
	}

	if len(where) > 0 {
		whereClause = ` WHERE ` + strings.Join(where, ` AND `)
	}

	return sqBegin + whereClause + sqEnd
}

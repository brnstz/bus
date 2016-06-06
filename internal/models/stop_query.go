package models

import "strings"

const (
	sqBeginDistinct = `
		SELECT * FROM (
			SELECT
				DISTINCT ON (stop.route_id, direction_id)
				stop_id,
				stop_name,
				direction_id,
				headsign,
				stop.route_id,
				stop.agency_id,
				latitude(location) AS lat,
				longitude(location) AS lon,
				earth_distance(location, ll_to_earth(:mid_lat, :mid_lon)) 
					AS dist
				FROM stop  INNER JOIN 
					route ON stop.route_id = route.route_id
	`
	sqBegin = `
		SELECT
			stop_id,
			stop_name,
			direction_id,
			headsign,
			stop.route_id,
			stop.agency_id,
			latitude(location) AS lat,
			longitude(location) AS lon,
			earth_distance(location, ll_to_earth(:mid_lat, :mid_lon)) 
				AS dist
			FROM stop  INNER JOIN 
				route ON stop.route_id = route.route_id
	`

	sqLocationFilter = `
		earth_box(ll_to_earth(:mid_lat, :mid_lon), :dist) @> location
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

	sqEndDistinct = ` 
		ORDER BY stop.route_id, direction_id, dist 
	    ) unique_routes ORDER BY dist ASC LIMIT 10
	`
)

type StopQuery struct {
	/*
		// The southwest and northeast bounding points of the box we are
		// searching
		// FIXME: ignored for now, we need to use something other than
		// pg earthdistance (probably post_gis)
			SWLat float64 `db:"sw_lat"`
			SWLon float64 `db:"sw_lon"`
			NELat float64 `db:"ne_lat"`
			NELon float64 `db:"ne_lon"`
	*/

	// The midpoint of our search box
	MidLat float64 `db:"mid_lat"`
	MidLon float64 `db:"mid_lon"`

	// The distance in meters we're searching from the midpoint
	Dist float64 `db:"dist"`

	// filter on this specific RouteType if specified
	RouteType string `db:"-"`

	RouteTypeInt int `db:"route_type"`

	// append departures to returned stops
	Departures bool

	Distinct bool

	RouteID  string `db:"route_id"`
	AgencyID string `db:"agency_id"`

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

	if !(sq.MidLat == 0.0 && sq.MidLon == 0.0 && sq.Dist == 0.0) {
		where = append(where, sqLocationFilter)
	}

	if len(where) > 0 {
		whereClause = ` WHERE ` + strings.Join(where, ` AND `)
	}

	if sq.Distinct {
		return sqBeginDistinct + whereClause + sqEndDistinct
	} else {
		return sqBegin + whereClause
	}
}

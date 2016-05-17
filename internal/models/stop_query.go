package models

const (
	sqBegin = `
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
				WHERE 
					earth_box(ll_to_earth(:mid_lat, :mid_lon), :dist) @> 
						location
	`
	sqRouteFilter = ` 
					AND route.route_type = :route_type
	`
	sqEnd = ` 
				ORDER BY stop.route_id, direction_id, dist 
			) unique_routes ORDER BY dist ASC
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
	RouteType int `db:"route_type"`

	routeFilter bool
}

// NewStopQueryDist returns a StopQuery that searches a distance from
// a midpoint. If routeType is an empty strings, stops for all route stops
// will be returned.
func NewStopQueryDist(midLat, midLon, dist float64, routeType string) (*StopQuery, error) {

	sq := &StopQuery{
		MidLat: midLat,
		MidLon: midLon,
		Dist:   dist,
	}

	if len(routeType) > 0 {
		var ok bool
		sq.routeFilter = true
		sq.RouteType, ok = routeTypeInt[routeType]
		if !ok {
			return nil, ErrInvalidRouteType
		}
	}

	return sq, nil
}

// Query returns the SQL for this StopQuery
func (sq *StopQuery) Query() string {
	var query string

	if sq.routeFilter {
		query = sqBegin + sqRouteFilter + sqEnd
	} else {
		query = sqBegin + sqEnd
	}

	return query
}

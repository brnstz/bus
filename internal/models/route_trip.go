package models

import (
	"log"

	"github.com/brnstz/upsert"
	"github.com/jmoiron/sqlx"
)

// RouteTrip is a one-to-many mapping between routes and shapes.
// The loader should identify which shapes represent the typical / full
// route and create/save these objects
type RouteTrip struct {
	AgencyID    string `json:"agency_id" db:"agency_id" upsert:"key"`
	RouteID     string `json:"route_id" db:"route_id" upsert:"key"`
	Headsign    string `json:"headsign" db:"headsign" upsert:"key"`
	DirectionID int    `json:"direction_id" db:"direction_id" upsert:"key"`
	TripID      string `json:"trip_id" db:"trip_id"`

	Count int `json:"-" db:"count" upsert:"omit"`
}

// Table returns the name of the RouteShape table, implementing the
// upsert.Upserter interface
func (rt *RouteTrip) Table() string {
	return "route_trip"
}

// DeleteRouteTrips removes all existing shapes. Typically
// this should be used in a transaction in conjuction with GetRouteShapes
// to rebuild the data
func DeleteRouteTrips(db sqlx.Ext) error {
	_, err := db.Exec(`DELETE FROM route_trip`)
	if err != nil {
		log.Println("can't delete route_trip", err)
		return err
	}

	return nil
}

// GetRouteTrips returns distinct trip_ids order by the "size" (number of
// stops) of the route from least to most for each  given combination of
// agency/route/headsign/direction. Given the ordering,
// you can Save() each value in a tx and end up with the "best" value
// live in the db
func GetRouteTrips(db sqlx.Ext) ([]*RouteTrip, error) {
	rt := []*RouteTrip{}

	q := `
		SELECT count(*) AS count,
			   td.trip_id,
			   td.agency_id, td.route_id, td.headsign, td.direction_id
		FROM scheduled_stop_time sst INNER JOIN

		(SELECT DISTINCT trip_id, agency_id, route_id, headsign, direction_id
		 FROM trip
		) AS td ON td.trip_id = sst.trip_id

		GROUP BY td.trip_id, td.agency_id, td.route_id, td.headsign, 
			     td.direction_id

		ORDER BY count(*) ASC
	`

	err := sqlx.Select(db, &rt, q)
	if err != nil {
		log.Println("can't get route trips", err)
		return rt, err
	}

	return rt, nil
}

// Save saves the route_trip to the db
func (rt *RouteTrip) Save(db sqlx.Ext) error {
	_, err := upsert.Upsert(db, rt)
	return err
}

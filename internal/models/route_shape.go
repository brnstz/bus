package models

import (
	"github.com/brnstz/upsert"
	"github.com/jmoiron/sqlx"
)

// RouteShape is a one-to-many mapping between routes and shapes.
// The loader should identify the minimally
type RouteShape struct {
	AgencyID    string `db:"agency_id" upsert:"key"`
	RouteID     string `db:"route_id" upsert:"key"`
	Headsign    string `db:"headsign" upsert:"key"`
	DirectionID int    `db:"direction_id" upsert:"key"`
	ShapeID     string `db:"shape_id"`
	Count       int    `db:"count" upsert:"omit"`
}

func (rs *RouteShape) Table() string {
	return "route_shape"
}

// DeleteRouteShapes removes all existing shapes. Typically
// this should be used in a transaction in conjuction with GetRouteShapes
// to rebuild the data
func DeleteRouteShapes(db sqlx.Ext) error {
	_, err := db.Exec(`DELETE FROM route_shape`)

	return err
}

// GetRouteShapes returns distinct shapes for every route ordered by
// the "size" (number of points) of the route from least to most for each
// given combination of agency/route/headsign/direction. Given the ordering,
// you can Save() each value in a tx and end up with the "best" value
// live in the db
func GetRouteShapes(db sqlx.Ext) ([]*RouteShape, error) {
	rs := []*RouteShape{}

	q := `
   		SELECT count(*) AS count, td.agency_id, td.shape_id, 
			   td.route_id, td.headsign, td.direction_id
        FROM shape INNER JOIN 

		(SELECT DISTINCT shape_id, agency_id, route_id, headsign, direction_id
         FROM trip
        ) AS td ON shape.shape_id = td.shape_id

        GROUP BY td.shape_id, td.agency_id, 
				 td.route_id, td.headsign, td.direction_id

		ORDER BY count(*) ASC
	`

	err := sqlx.Select(db, &rs, q)

	return rs, err
}

// Save saves the route_shape to the db
func (rs *RouteShape) Save(db sqlx.Ext) error {
	_, err := upsert.Upsert(db, rs)
	return err
}

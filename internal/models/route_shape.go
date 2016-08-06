package models

import (
	"log"

	"github.com/brnstz/upsert"
	"github.com/jmoiron/sqlx"
)

// RouteShape is a one-to-many mapping between routes and shapes.
// The loader should identify which shapes represent the typical / full
// route and create/save these objects
type RouteShape struct {
	AgencyID    string `json:"agency_id" db:"agency_id" upsert:"key"`
	RouteID     string `json:"route_id" db:"route_id" upsert:"key"`
	Headsign    string `json:"headsign" db:"headsign" upsert:"key"`
	DirectionID int    `json:"direction_id" db:"direction_id" upsert:"key"`
	ShapeID     string `json:"shape_id" db:"shape_id"`

	Count  int    `json:"-" db:"count" upsert:"omit"`
	TripID string `json:"-" db:"trip_id" upsert:"omit"`

	Shapes []*Shape `json:"shapes" db:"-" upsert:"omit"`
}

// Table returns the name of the RouteShape table, implementing the
// upsert.Upserter interface
func (rs *RouteShape) Table() string {
	return "route_shape"
}

// DeleteRouteShapes removes all existing route / fake shapes. Typically
// this should be used in a transaction in conjuction with GetRouteShapes
// to rebuild the data
func DeleteRouteShapes(db sqlx.Ext) error {
	_, err := db.Exec(`DELETE FROM route_shape`)
	if err != nil {
		log.Println("can't delete route_shape", err)
		return err
	}

	return nil
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
	if err != nil {
		log.Println("can't get route shapes", err)
		return rs, err
	}

	return rs, nil
}

func addRevCopies(rs []*RouteShape, from, to int) []*RouteShape {
	newShapes := []*RouteShape{}

	// copy all shapes into reverse version
	for _, oldShape := range rs {
		// make a copy of the shape
		shape := *oldShape

		// set opposite direction
		shape.DirectionID = to

		// add opposite points
		size := len(oldShape.Shapes)
		shape.Shapes = make([]*Shape, size)
		for i := 0; i < size; i++ {
			shape.Shapes[i] = oldShape.Shapes[size-i-1]
		}

		newShapes = append(newShapes, &shape)
	}

	return newShapes
}

// GetSavedRouteShapes returns all shapes for this combination of
// agencyID and routeID
func GetSavedRouteShapes(db sqlx.Ext, agencyID, routeID string) ([]*RouteShape, error) {
	rs := []*RouteShape{}

	q := `
		SELECT *
		FROM route_shape
		WHERE agency_id = $1 AND
		      route_id  = $2
	`

	err := sqlx.Select(db, &rs, q, agencyID, routeID)
	if err != nil {
		log.Println("can't get saved route shapes", err)
		return rs, err
	}

	// found01 ensures we find both direction_id 0 and direction_id 1
	// (the values 0 and 1 are part of the spec, not a special case)
	found01 := map[int]bool{}

	for _, shape := range rs {
		found01[shape.DirectionID] = true
		shape.Shapes, err = GetShapes(db, agencyID, shape.ShapeID)
		if err != nil {
			log.Println("can't get shapes", err)
			return rs, err
		}
	}

	// If we can't find either, then there's nothing else we can do
	if !found01[0] && !found01[1] {
		return rs, err
	}

	// If we can't find 0, then copy a reverse version of 1
	if !found01[0] {
		rs = append(rs, addRevCopies(rs, 1, 0)...)
	} else if !found01[1] {
		rs = append(rs, addRevCopies(rs, 0, 1)...)
	}

	return rs, err
}

// Save saves the route_shape to the db
func (rs *RouteShape) Save(db sqlx.Ext) error {
	_, err := upsert.Upsert(db, rs)
	return err
}

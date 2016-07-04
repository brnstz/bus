package models

import (
	"log"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
	"github.com/jmoiron/sqlx"
)

type Trip struct {
	AgencyID string `json:"agency_id" db:"agency_id" upsert:"key"`
	RouteID  string `json:"route_id" db:"route_id" upsert:"key"`
	ID       string `json:"trip_id" db:"trip_id" upsert:"key"`

	ServiceID string `json:"service_id" db:"service_id"`
	ShapeID   string `json:"shape_id" db:"shape_id"`

	Headsign    string `json:"headsign" db:"headsign"`
	DirectionID int    `json:"direction_id" db:"direction_id"`

	ShapePoints []*Shape `json:"shape_points" db:"-" upsert:"omit"`
	Stops       []*Stop  `json:"stops" db:"-" upsert:"omit"`
}

func NewTrip(id, routeID, agencyID, serviceID, shapeID, headsign string, direction int) (t *Trip, err error) {
	t = &Trip{
		ID:          id,
		AgencyID:    agencyID,
		RouteID:     routeID,
		ServiceID:   serviceID,
		ShapeID:     shapeID,
		Headsign:    headsign,
		DirectionID: direction,
	}

	return
}

func (t *Trip) Table() string {
	return "trip"
}

// Save saves a trip to the database
func (t *Trip) Save() error {
	_, err := upsert.Upsert(etc.DBConn, t)
	return err
}

func (t *Trip) addShapes(db sqlx.Ext, agencyID, shapeID string) (err error) {
	// Try to get the shapes specific to this trip
	q := `
		SELECT 
			ST_X(location::geometry) AS lat, 
			ST_Y(location::geometry) AS lon
		FROM shape
		WHERE agency_id = $1 AND
		      shape_id  = $2
		ORDER BY seq ASC
	`

	err = etc.DBConn.Select(&t.ShapePoints, q, agencyID, shapeID)
	if err != nil {
		log.Println("can't get shapes", err)
		return
	}

	// If we got some points, we're good
	if len(t.ShapePoints) > 0 {
		return
	}

	// But shapes are optional for trips. If there aren't any, try to get one
	// for the route.
	routeShapes, err := GetSavedRouteShapes(db, agencyID, t.RouteID)
	if err != nil {
		log.Println("can't get saved route shapes", err)
		return
	}

	// At least we want a route shape that matches our direction
	var directionRouteShape *RouteShape

	// Ideally we get one that also matches the headsign
	var headsignShape *RouteShape

	for _, rs := range routeShapes {
		if rs.DirectionID == t.DirectionID {
			directionRouteShape = rs

			if rs.Headsign == t.Headsign {
				headsignShape = rs
			}
		}
	}

	if headsignShape != nil {
		t.ShapePoints = headsignShape.Shapes
		t.ShapeID = headsignShape.ShapeID

	} else if directionRouteShape != nil {
		t.ShapePoints = directionRouteShape.Shapes
		t.ShapeID = directionRouteShape.ShapeID

	} else {
		// FIXME: final backup: draw a line from stop to stop?
		log.Println("can't get shape for", t)
	}

	return
}

// GetTrip returns the trip for this agency and trip ID
func GetTrip(db sqlx.Ext, agencyID, routeID, tripID string) (t Trip, err error) {

	// Get the trip
	q := `
		SELECT * 
		FROM trip 
		WHERE agency_id	= $1 AND
		      trip_id = $2 AND 
			  route_id = $3
	`

	err = sqlx.Get(db, &t, q, agencyID, tripID, routeID)
	if err != nil {
		log.Println("can't get trip", q, agencyID, tripID, routeID, err)
		return
	}

	err = t.addShapes(db, agencyID, t.ShapeID)
	if err != nil {
		log.Println("can't get shapes", err)
		return
	}

	t.Stops, err = GetStopsByTrip(db, &t)
	if err != nil {
		log.Println("can't get trip stops", err)
		return
	}

	return
}

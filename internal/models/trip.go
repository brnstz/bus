package models

import (
	"log"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
)

type Trip struct {
	AgencyID string `json:"agency_id" db:"agency_id" upsert:"key"`
	RouteID  string `json:"route_id" db:"route_id" upsert:"key"`
	ID       string `json:"trip_id" db:"trip_id" upsert:"key"`

	ServiceID string `json:"service_id" db:"service_id"`
	ShapeID   string `json:"shape_id" db:"shape_id"`

	Headsign    string `json:"headsign" db:"headsign"`
	DirectionID int    `json:"direction_id" db:"direction_id"`

	ShapePoints []struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"shape_points" db:"-" upsert:"omit"`
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

func (t *Trip) addShapes(agencyID, shapeID string) (err error) {
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

	return
}

func GetAnyTrip(agencyID, routeID string) (t Trip, err error) {
	var dummy int
	var shapeID string
	var tripID string

	// Get the most "popular" route
	q := `
		SELECT count(*) AS cnt, shape_id, trip_id
		FROM trip
		WHERE route_id = $1 AND char_length(shape_id) > 0
		GROUP BY shape_id, trip_id
		ORDER BY cnt DESC
		LIMIT 1
	`
	row := etc.DBConn.QueryRowx(q, routeID)
	err = row.Scan(&dummy, &shapeID, &tripID)
	if err != nil {
		log.Println("can't get shape_id", err)
		return
	}

	// Get the trip
	q = `
		SELECT * 
		FROM trip 
		WHERE agency_id	= $1 AND
		      trip_id = $2 AND 
			  route_id = $3
	`

	err = etc.DBConn.Get(&t, q, agencyID, tripID, routeID)
	if err != nil {
		log.Println("can't get trip", q, agencyID, tripID, routeID, err)
		return
	}

	err = t.addShapes(agencyID, shapeID)
	if err != nil {
		log.Println("can't get shape", err)
		return
	}

	return
}

// GetTrip returns the trip for this agency and trip ID
func GetTrip(agencyID, routeID, tripID string) (t Trip, err error) {

	// Get the trip
	q := `
		SELECT * 
		FROM trip 
		WHERE agency_id	= $1 AND
		      trip_id = $2 AND 
			  route_id = $3
	`

	err = etc.DBConn.Get(&t, q, agencyID, tripID, routeID)
	if err != nil {
		//log.Println("can't get trip", q, agencyID, tripID, routeID, err)
		return
	} else {

		err = t.addShapes(agencyID, t.ShapeID)
		if err != nil {
			log.Println("can't get shapes", err)
			return
		}

		if len(t.ShapePoints) > 0 {
			return
		}
	}

	return
}

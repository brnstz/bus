package models

import (
	"log"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
)

type Trip struct {
	AgencyID string `json:"agency_id" db:"agency_id" upsert:"key"`
	ID       string `json:"trip_id" db:"trip_id" upsert:"key"`

	ServiceID string `json:"service_id" db:"service_id"`
	ShapeID   string `json:"shape_id" db:"shape_id"`

	Headsign    string `json:"-" db:"-" upsert:"omit"`
	DirectionID int    `json:"-" db:"-" upsert:"omit"`

	ShapePoints []struct {
		Lat float64
		Lon float64
	} `json:"shape_points" db:"-"`
}

func NewTrip(id, agencyID, serviceID, shapeID, headsign string, direction int) (t *Trip, err error) {
	t = &Trip{
		ID:          id,
		AgencyID:    agencyID,
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

func GetTrip(agencyID string, tripID string) (t *Trip, err error) {
	q := `
		SELECT * 
		FROM trip 
		WHERE agency_id	= $1 AND
		      trip_id = $2
	`

	err = etc.DBConn.Select(t, q, agencyID, tripID)
	if err != nil {
		log.Println("can't get trip", err)
		return
	}

	q = `
		SELECT 
			latitude(location) AS lat,
			longitude(location) AS lon
		FROM shape
		WHERE agency_id = $1 AND
		      shape_id  = $2
		ORDER BY seq ASC
	`

	err = etc.DBConn.Select(t.ShapePoints, q, agencyID, t.ShapeID)
	if err != nil {
		log.Println("can't get shapes", err)
		return
	}

	return
}

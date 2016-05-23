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
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"shape_points" db:"-" upsert:"omit"`
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

// GetTrip returns the trip for this agency and trip ID
func GetTrip(agencyID, routeID, tripID string) (t Trip, err error) {

	// Get the trip
	q := `
		SELECT * 
		FROM trip 
		WHERE agency_id	= $1 AND
		      trip_id = $2
	`

	err = etc.DBConn.Get(&t, q, agencyID, tripID)
	if err != nil {
		log.Println("can't get trip")
		return
	}

	// Try to get the shapes specific to this trip
	q = `
		SELECT 
			latitude(location) AS lat,
			longitude(location) AS lon
		FROM shape
		WHERE agency_id = $1 AND
		      shape_id  = $2
		ORDER BY seq ASC
	`

	err = etc.DBConn.Select(&t.ShapePoints, q, agencyID, t.ShapeID)
	if err != nil {
		log.Println("can't get shapes", err)
		return
	}

	/*
		FIXME
			// If we got the shapes, then stop
			if err == nil {
				return
			}

			now := time.Now()

			// Otherwise, we need to try getting by route id
			todayName := strings.ToLower(now.Format("Monday"))

			serviceID, err := getServiceIDByDay(etc.DBConn, routeID, todayName, now)
	*/

	return
}

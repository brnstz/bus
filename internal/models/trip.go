package models

import (
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

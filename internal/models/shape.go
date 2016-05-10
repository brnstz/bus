package models

import (
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
)

type Shape struct {
	ID       string `json:"shape_id" db:"shape_id" upsert:"key"`
	AgencyID string `json:"agency_id" db:"agency_id" upsert:"key"`
	Seq      int    `json:"seq" db:"seq" upsert:"key"`

	Lat float64 `json:"lat" db:"lat" upsert:"omit"`
	Lon float64 `json:"lon" db:"lon" upsert:"omit"`

	// Location is an "earth" field value that combines lat and lon into
	// a single field.
	Location interface{} `json:"-" db:"location" upsert_value:"ll_to_earth(:lat, :lon)"`
}

func NewShape(id, agencyID string, seq int, lat, lon float64) (s *Shape, err error) {
	s = &Shape{
		ID:       id,
		AgencyID: agencyID,
		Seq:      seq,
		Lat:      lat,
		Lon:      lon,
	}

	return
}

func (s *Shape) Table() string {
	return "shape"
}

// Save saves a shape to the database
func (s *Shape) Save() error {
	_, err := upsert.Upsert(etc.DBConn, s)
	return err
}

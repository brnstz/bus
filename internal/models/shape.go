package models

import (
	"log"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
	"github.com/jmoiron/sqlx"
)

type Shape struct {
	ID       string `json:"-" db:"shape_id" upsert:"key"`
	AgencyID string `json:"-" db:"agency_id" upsert:"key"`
	Seq      int    `json:"-" db:"seq" upsert:"key"`

	Lat float64 `json:"lat" db:"lat" upsert:"omit"`
	Lon float64 `json:"lon" db:"lon" upsert:"omit"`

	// Location is PostGIS field value that combines lat and lon into a single
	// field.
	Location interface{} `json:"-" db:"location" upsert_value:"ST_SetSRID(ST_MakePoint(:lat, :lon),4326)"`
}

func GetShapes(db sqlx.Ext, agencyID, shapeID string) ([]*Shape, error) {
	shapes := []*Shape{}
	if conf.API.LogTiming {
		t1 := time.Now()
		defer func() { log.Println(time.Now().Sub(t1)) }()
	}

	q := `
		SELECT ST_X(location) AS lat, ST_Y(location) AS lon
		FROM shape
		WHERE 
			agency_id = $1 AND
			shape_id = $2
		ORDER BY seq ASC
	`
	err := sqlx.Select(db, &shapes, q, agencyID, shapeID)
	if err != nil {
		log.Println("can't get shapes", err)
		return shapes, err
	}

	return shapes, nil
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

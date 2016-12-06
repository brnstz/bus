package models

import (
	"log"

	null "gopkg.in/guregu/null.v3"

	"github.com/brnstz/upsert"
	"github.com/jmoiron/sqlx"
)

type Shape struct {
	ID       string `json:"-" db:"shape_id" upsert:"key"`
	AgencyID string `json:"-" db:"agency_id" upsert:"key"`
	Seq      int    `json:"-" db:"seq" upsert:"key"`

	Lat null.Float `json:"lat" db:"lat"`
	Lon null.Float `json:"lon" db:"lon"`

	// Location is PostGIS field value that combines lat and lon into a single
	// field.
	Location interface{} `json:"-" db:"location" upsert_value:"ST_SetSRID(ST_MakePoint(:lat, :lon),4326)"`
}

func GetShapes(db sqlx.Ext, agencyID, shapeID string) ([]*Shape, error) {
	shapes := []*Shape{}

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
	}
	s.Lat.Scan(lat)
	s.Lon.Scan(lon)

	return
}

func (s *Shape) Table() string {
	return "shape"
}

// Save saves a shape to the database
func (s *Shape) Save(db sqlx.Ext) error {
	_, err := upsert.Upsert(db, s)
	return err
}

package models

import (
	"fmt"
	"log"

	"github.com/brnstz/upsert"
	"github.com/jmoiron/sqlx"
)

type FakeShape struct {
	AgencyID    string `json:"-" db:"agency_id" upsert:"key"`
	RouteID     string `json:"-" db:"route_id" upsert:"key"`
	DirectionID int    `json:"-" db:"direction_id" upsert:"key"`
	Headsign    string `json:"-" db:"headsign" upsert:"key"`
	Seq         int    `json:"-" db:"seq" upsert:"key"`

	Lat float64 `json:"lat" db:"lat" upsert:"omit"`
	Lon float64 `json:"lon" db:"lon" upsert:"omit"`

	// Location is PostGIS field value that combines lat and lon into a single
	// field.
	Location interface{} `json:"-" db:"location" upsert_value:"ST_SetSRID(ST_MakePoint(:lat, :lon),4326)"`

	TripID  string `json:"-" db:"trip_id" upsert:"omit"`
	ShapeID string `json:"-" db:"trip_id" upsert:"omit"`
}

func (s *FakeShape) Table() string {
	return "fake_shape"
}

// Save saves a fake shape to the database
func (fs *FakeShape) Save(db sqlx.Ext) error {
	_, err := upsert.Upsert(db, fs)
	return err
}

// DeleteFakeShapes removes all existing fake shapes. Typically
// this should be used in a transaction in conjuction with
// to rebuild the data via GetFakeShapes
func DeleteFakeShapes(db sqlx.Ext) error {

	_, err := db.Exec(`DELETE FROM fake_shape`)
	if err != nil {
		log.Println("can't delete fake_shape", err)
		return err
	}

	return nil
}

// GetFakeRouteShapes returns fake shapes for agency/route/headsign/direction
// combos that don't have a shape
func GetFakeRouteShapes(db sqlx.Ext) ([]*FakeShape, error) {
	var fs FakeShape

	yesShape := map[string]FakeShape{}
	noShape := map[string]FakeShape{}

	shapes := []*FakeShape{}

	// Get all combinations of agency/route/headsign/dir/shape
	q := `
		SELECT DISTINCT agency_id, route_id, headsign, shape_id, direction_id
		FROM trip 
	`

	rows, err := db.Queryx(q)
	if err != nil {
		log.Println("can't get missing shapes", err)
		return shapes, err
	}

	// If there is an empty shape for this combo, then record in noShape,
	// otherwise in yesShape
	for rows.Next() {
		err = rows.Scan(&fs.AgencyID, &fs.RouteID, &fs.Headsign,
			&fs.ShapeID, &fs.DirectionID)
		if err != nil {
			log.Println("can't scan missing route shape", err)
			return shapes, err
		}

		id := fmt.Sprintf("%s|%s|%s|%d",
			fs.AgencyID, fs.RouteID, fs.Headsign, fs.DirectionID,
		)
		if len(fs.ShapeID) < 1 {
			noShape[id] = fs
		} else {
			yesShape[id] = fs
		}
	}

	// Go through all values in noShape
	for k, v := range noShape {

		theseShapes := []*FakeShape{}

		// Ignore if there is also a yesShape
		_, exists := yesShape[k]
		if exists {
			continue
		}

		log.Println("need a fake shape for", v)

		// Otherwise we're good to continue

		// Get the trip ID that has most points to represent for
		// missing shape
		err = sqlx.Get(db, &fs, `
			SELECT COUNT(*) AS cnt,
				   trip.agency_id, trip.route_id, trip.headsign,
				   trip.direction_id, trip.trip_id

			FROM trip

			INNER JOIN scheduled_stop_time sst ON
							sst.agency_id = trip.agency_id AND
							sst.route_id  = trip.route_id  AND
							sst.trip_id   = trip.trip_id

			WHERE trip.agency_id    = $1 AND
				  trip.route_id     = $2 AND
				  trip.headsign     = $3 AND
				  trip.direction_id = $4

			GROUP BY trip.agency_id, trip.route_id, trip.headsign,
					 trip.direction_id, trip.trip_id

			ORDER BY count(*) DESC
			LIMIT 1
		`, v.AgencyID, v.RouteID, v.Headsign, v.DirectionID)

		if err != nil {
			log.Println("can't get trip id")
			return shapes, err
		}

		// Now get all stops for that trip as a FakeShape object
		err = sqlx.Select(db, &theseShapes, `
			SELECT 
				sst.stop_sequence, ST_X(stop.location) AS lat, ST_Y(stop.location) AS lon,
				trip.agency_id, trip.route_id, trip.headsign,
			    trip.direction_id, trip.trip_id

			FROM trip

			INNER JOIN scheduled_stop_time sst ON
				sst.agency_id = trip.agency_id AND
				sst.route_id  = trip.route_id  AND
				sst.trip_id   = trip.trip_id

			INNER JOIN stop ON 
				sst.agency_id = stop.agency_id AND
				sst.route_id  = stop.route_id  AND
				sst.stop_id   = stop.stop_id 

			WHERE trip.agency_id    = $1 AND
				  trip.route_id     = $2 AND
				  trip.headsign     = $3 AND
				  trip.direction_id = $4
		`, v.AgencyID, v.RouteID, v.Headsign, v.DirectionID)

		if err != nil {
			log.Println("can't get trip id", err)
			return shapes, err
		}

		shapes = append(shapes, theseShapes...)
	}

	return shapes, nil
}

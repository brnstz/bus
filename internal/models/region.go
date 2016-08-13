package models

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

func getRegionAgencyIDs(db sqlx.Ext, swlat, swlon, nelat, nelon float64) (agencyIDs []string, err error) {
	ls := fmt.Sprintf(
		`LINESTRING(%f %f, %f %f, %f %f, %f %f, %f %f)`,
		swlat, swlon,
		swlat, nelon,
		nelat, nelon,
		nelat, swlon,
		swlat, swlon,
	)

	q := `
		SELECT agency_id
		FROM region
		WHERE ST_INTERSECTS(
				ST_SETSRID(
					ST_MAKEPOLYGON($1), 4326), locations)
	`

	err = sqlx.Select(db, &agencyIDs, q, ls)
	if err != nil {
		log.Println("can't get regions", err)
		return
	}

	return
}

package models

import (
	"log"

	"github.com/brnstz/bus/internal/etc"
	"github.com/jmoiron/sqlx"
)

func GetRegions(db sqlx.Ext, swlat, swlon, nelat, nelon float64) (regions []string, err error) {
	ls := etc.BoundsToLineString(swlat, swlon, nelat, nelon)

	q := `
		SELECT DISTINCT region_id
		FROM region
		WHERE ST_INTERSECTS(
				ST_SETSRID(
					ST_MAKEPOLYGON($1), 4326), locations)
	`

	err = sqlx.Select(db, &regions, q, ls)
	if err != nil {
		log.Println("can't get regions", err)
		return
	}

	return
}

func getRegionAgencyIDs(db sqlx.Ext, swlat, swlon, nelat, nelon float64) (agencyIDs []string, err error) {
	ls := etc.BoundsToLineString(swlat, swlon, nelat, nelon)

	q := `
		SELECT agency_id
		FROM region
		WHERE ST_INTERSECTS(
				ST_SETSRID(
					ST_MAKEPOLYGON($1), 4326), locations)
	`

	err = sqlx.Select(db, &agencyIDs, q, ls)
	if err != nil {
		log.Println("can't get region agency ids", err)
		return
	}

	return
}

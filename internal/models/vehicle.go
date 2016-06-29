package models

import (
	"log"

	"github.com/brnstz/bus/internal/etc"
	"github.com/jmoiron/sqlx"
)

// Vehicle is the location (and maybe any other info) we want to provide
// about a
type Vehicle struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`

	// Is this location live or estimated based on scheduled?
	Live bool `json:"live"`
}

func GetVehicle(agencyID, routeID, stopID string, directionID int) (vehicle Vehicle, err error) {
	q := `
		SELECT latitude(location) AS lat, longitude(location) AS lon
		FROM   stop
		WHERE  agency_id    = $1 AND
			   route_id     = $2 AND
			   stop_id      = $3 AND
			   direction_id = $4
	`

	err = sqlx.Get(etc.DBConn, &vehicle, q, agencyID, routeID, stopID,
		directionID)
	if err != nil {
		log.Println("can't get vehicle", err)
		return
	}

	return
}

type vehicleRes struct {
	StopID       string `db:"stop_id"`
	TripID       string `db:"trip_id"`
	DepartureSec int    `db:"departure_sec"`
}

func getVehicles(agencyID, routeID string, directionID int, serviceIDs []string, minSec int) (vehicles []Vehicle, err error) {
	var vehicle Vehicle

	for _, serviceID := range serviceIDs {

		// For each service id, get all departures. The first one for each
		// trip_id is the actual stop where the train is (estimated to be)
		// at.

		vr := []vehicleRes{}

		q := `
            SELECT sst.stop_id, sst.trip_id, sst.departure_sec
            FROM scheduled_stop_time sst
		    INNER JOIN trip ON sst.agency_id = trip.agency_id AND
                               sst.route_id  = trip.route_id  AND
                               sst.trip_id   = trip.trip_id	
            INNER JOIN (
                SELECT max(departure_sec) AS max_departure, agency_id, trip_id, route_id
                FROM scheduled_stop_time max_sst_inner
                GROUP BY agency_id, trip_id, route_id
            ) max_sst ON sst.agency_id = max_sst.agency_id AND
                         sst.route_id  = max_sst.route_id AND
                         sst.trip_id   = max_sst.trip_id

            WHERE sst.agency_id          = $1 AND
                  sst.route_id           = $2 AND
                  sst.service_id         = $3 AND
                  sst.departure_sec     <= $4 AND
                  max_sst.max_departure >= $4 AND
                  trip.direction_id      = $5
            ORDER BY trip_id, departure_sec DESC 
	    `

		err = sqlx.Select(etc.DBConn, &vr, q, agencyID, routeID, serviceID, minSec, directionID)
		if err != nil {
			log.Println("can't get vehicle results", err)
			return
		}

		// Get the first stop for each trip, and retrieve lat/lon
		lastTripID := ""
		for _, res := range vr {
			// Ignore until we get a new trip
			if lastTripID == res.TripID {
				continue
			}

			lastTripID = res.TripID

			vehicle, err = GetVehicle(agencyID, routeID, res.StopID, directionID)
			if err != nil {
				log.Println("can't get vehicle", err)
				return
			}

			vehicles = append(vehicles, vehicle)
		}
	}

	return
}

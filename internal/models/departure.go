package models

import (
	"log"
	"time"

	"github.com/brnstz/bus/internal/etc"
)

const (
	// midnightSecs is what a depature_sec value of midnight looks like. We
	// use this for trying to find departures after midnight.
	midnightSecs = 86400

	// maximum departures to return in a result
	MaxDepartures = 5
)

type Departure struct {
	Time         time.Time `json:"time" db:"-"`
	DepartureSec int       `json:"-" db:"departure_sec"`
	TripID       string    `json:"trip_id" db:"trip_id"`
	ServiceID    string    `json:"service_id" db:"service_id"`
	Live         bool      `json:"live" db:"-" upsert:"omit"`

	baseTime time.Time `json:"-" db:"-"`
}

func (d *Departure) Initialize() error {
	d.Time = d.baseTime.Add(time.Second * time.Duration(d.DepartureSec))

	return nil
}

type SortableDepartures []*Departure

func (d SortableDepartures) Len() int {
	return len(d)
}

func (d SortableDepartures) Less(i, j int) bool {
	return d[i].Time.Before(d[j].Time)
}

func (d SortableDepartures) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func getDepartures(agencyID, routeID, stopID, serviceID string, minSec int, base time.Time) (d []*Departure, err error) {
	q := `
		SELECT departure_sec, trip_id, service_id
		FROM   scheduled_stop_time
		WHERE  
			   agency_id  =	$1 AND
			   route_id   =	$2 AND
			   stop_id    =	$3 AND
			   service_id =	$4 AND

			   departure_sec    >= $5

		ORDER BY departure_sec LIMIT $6
	`

	err = etc.DBConn.Select(&d, q, agencyID, routeID, stopID, serviceID,
		minSec, MaxDepartures)

	if err != nil {
		log.Println("can't get departures", err)
		return
	}

	for _, departure := range d {
		departure.baseTime = base
		err = departure.Initialize()
		if err != nil {
			log.Println("can't init departure", err)
			return
		}
	}

	return
}

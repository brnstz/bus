package models

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
)

type ScheduledStopTime struct {
	AgencyID  string `db:"agency_id" upsert:"key"`
	RouteID   string `db:"route_id" upsert:"key"`
	StopID    string `db:"stop_id" upsert:"key"`
	ServiceID string `db:"service_id" upsert:"key"`
	TripID    string `db:"trip_id" upsert:"key"`

	ArrivalSec   int `db:"arrival_sec"`
	DepartureSec int `db:"departure_sec"`

	StopSequence int `db:"stop_sequence"`

	LastStop sql.NullBool `db:"last_stop"`

	NextStopID  sql.NullString `db:"next_stop_id"`
	NextStopLat float64        `json:"next_stop_lat" db:"next_stop_lat" upsert:"omit"`
	NextStopLon float64        `json:"next_stop_lon" db:"next_stop_lon" upsert:"omit"`

	// Location is PostGIS field value that combines lat and lon into a single
	// field.
	NextStopLocation interface{} `json:"-" db:"next_stop_location" upsert_value:"ST_SetSRID(ST_MakePoint(:next_stop_lat, :next_stop_lon),4326)"`
}

func NewScheduledStopTime(routeID, stopID, serviceID, arrivalStr, depatureStr, agencyID, tripID string, sequence int, lastStop bool) (sst *ScheduledStopTime, err error) {
	asec := etc.TimeStrToSecs(arrivalStr)
	dsec := etc.TimeStrToSecs(depatureStr)

	sst = &ScheduledStopTime{
		RouteID:      routeID,
		StopID:       stopID,
		ServiceID:    serviceID,
		ArrivalSec:   asec,
		DepartureSec: dsec,
		AgencyID:     agencyID,
		TripID:       tripID,
		StopSequence: sequence,
	}

	err = sst.LastStop.Scan(lastStop)
	if err != nil {
		log.Println("can't scan last stop value")
		return
	}

	return
}

func (sst *ScheduledStopTime) Table() string {
	return "scheduled_stop_time"
}

// Save saves a scheduled_stop_time to the database
func (sst *ScheduledStopTime) Save() error {
	_, err := upsert.Upsert(etc.DBConn, sst)
	return err
}

func (s *ScheduledStopTime) String() string {
	return fmt.Sprintf("{%v %v %v @ %v (%v)}",
		s.RouteID, s.ServiceID, s.StopID,
		etc.SecsToTimeStr(s.DepartureSec), s.DepartureSec,
	)
}

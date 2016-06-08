package models

import (
	"fmt"

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
}

func NewScheduledStopTime(routeID, stopID, serviceID, arrivalStr, depatureStr, agencyID, tripID string, sequence int) (sst *ScheduledStopTime, err error) {
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

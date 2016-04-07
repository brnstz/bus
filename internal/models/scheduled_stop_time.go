package models

import (
	"fmt"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
)

type ScheduledStopTime struct {
	RouteID      string `db:"route_id" upsert:"key"`
	StopID       string `db:"stop_id" upsert:"key"`
	ServiceID    string `db:"service_id" upsert:"key"`
	DepartureSec int    `db:"departure_sec" upsert:"key"`
}

func NewScheduledStopTime(routeID, stopID, serviceID, timeStr string) (sst *ScheduledStopTime, err error) {
	dsec := etc.TimeStrToSecs(timeStr)

	sst = &ScheduledStopTime{
		RouteID:      routeID,
		StopID:       stopID,
		ServiceID:    serviceID,
		DepartureSec: dsec,
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

package models

import (
	"fmt"

	"github.com/brnstz/bus/internal/etc"
)

type ScheduledStopTime struct {
	RouteId      string `db:"route_id"`
	StopId       string `db:"stop_id"`
	ServiceId    string `db:"service_id"`
	DepartureSec int    `db:"departure_sec"`
}

func NewScheduledStopTime(routeId, stopId, serviceId, timeStr string) (sst ScheduledStopTime, err error) {
	dsec := etc.TimeStrToSecs(timeStr)

	sst = ScheduledStopTime{
		RouteId:      routeId,
		StopId:       stopId,
		ServiceId:    serviceId,
		DepartureSec: dsec,
	}

	return
}
func (s ScheduledStopTime) String() string {
	return fmt.Sprintf("{%v %v %v @ %v (%v)}",
		s.RouteId, s.ServiceId, s.StopId,
		etc.SecsToTimeStr(s.DepartureSec), s.DepartureSec,
	)
}

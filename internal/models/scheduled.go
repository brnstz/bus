package models

import (
	"fmt"
	"log"
	"time"

	"github.com/brnstz/bus/internal/etc"
	"github.com/jmoiron/sqlx"
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

// FIXME: is this correct?
func getServiceIdByDay(db sqlx.Ext, routeId, day string, now *time.Time) (serviceId string, err error) {
	row := db.QueryRowx(`
		SELECT service_id, route_id, max(start_date)
		FROM   service_route_day
		WHERE  day         = $1 AND
		       end_date    > $2 AND
			   route_id    = $3
		GROUP BY service_id, route_id
		LIMIT 1
	`, day, now, routeId,
	)

	var dummy1 string
	var dummy2 time.Time

	err = row.Scan(&serviceId, &dummy1, &dummy2)
	if err != nil {
		log.Println("can't scan service id", err, day, now, routeId)
		return
	}

	return
}

type ServiceRouteException struct {
	ServiceId     string
	RouteId       string
	ExceptionDate time.Time
}

type Departure struct {
	Time time.Time `json:"time" db:"time"`
	Desc string    `json:"desc"`

	// FIXME: stops away? miles away?
}

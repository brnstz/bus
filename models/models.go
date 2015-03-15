package models

import (
	"fmt"

	"github.com/brnstz/bus/common"
	"github.com/jmoiron/sqlx"
)

type Trip struct {
	Id          string
	Headsign    string
	DirectionId int
}

type Service struct {
	Id      string
	RouteId string
}

type ScheduledStopTime struct {
	RouteId      string `db:"route_id"`
	StopId       string `db:"stop_id"`
	ServiceId    string `db:"service_id"`
	DepartureSec int    `db:"departure_sec"`
}

func NewScheduledStopTime(routeId, stopId, serviceId, timeStr string) (sst ScheduledStopTime, err error) {
	dsec := common.TimeStrToSecs(timeStr)

	sst = ScheduledStopTime{
		RouteId:      routeId,
		StopId:       stopId,
		ServiceId:    serviceId,
		DepartureSec: dsec,
	}

	return
}
func (s ScheduledStopTime) String() string {
	return fmt.Sprintf("{%v %v %v @ %v (%v)}", s.RouteId, s.ServiceId, s.StopId, common.SecsToTimeStr(s.DepartureSec), s.DepartureSec)
}

type Stop struct {
	Id          string `json:"stop_id" db:"stop_id"`
	Name        string `json:"stop_name" db:"stop_name"`
	RouteId     string `json:"route_id" db:"route_id"`
	StationType string `json:"station_type" db:"stype"`

	DirectionId int    `json:"direction_id" db:"direction_id"`
	Headsign    string `json:"headsign" db:"headsign"`

	Lat float64 `json:"lat" db:"lat"`
	Lon float64 `json:"lon" db:"lon"`

	Dist float64 `json:"dist" db:"dist"`
}

func (s Stop) String() string {
	return fmt.Sprintf("{%v %v %v %v @ (%v,%v)}", s.Id, s.Name, s.RouteId, s.Headsign, s.Lat, s.Lon)
}

func (s Stop) Key() string {
	return fmt.Sprintf("%v%v", s.Id, s.RouteId)
}

func GetStopsByLoc(db sqlx.Ext, lat, lon, meters float64, filter string) ([]*Stop, error) {
	stops := []*Stop{}
	params := []interface{}{lat, lon, lat, lon, meters}

	q := `
		SELECT 
			stop_id,
			stop_name,
			direction_id,
			headsign,
			route_id,
			stype,
			latitude(location) AS lat,
			longitude(location) AS lon,
			earth_distance(location, ll_to_earth($1, $2)) AS dist 
		FROM stop 
		WHERE earth_box(ll_to_earth($3, $4), $5) @> location
	`
	if len(filter) > 0 {
		q = q + ` AND stype = $6 `
		params = append(params, filter)
	}

	q = q + ` ORDER BY dist ASC `

	err := sqlx.Select(db, &stops, q, params...)

	return stops, err
}

package models

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/brnstz/bus/common"
	"github.com/jmoiron/sqlx"
)

const maxStops = 3

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

func df(t time.Time) string {
	return t.Format("2006-01-02")
}

type timeSlice []time.Time

func (p timeSlice) Len() int {
	return len(p)
}

func (p timeSlice) Less(i, j int) bool {
	return p[i].Before(p[j])
}

func (p timeSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
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
	return fmt.Sprintf("{%v %v %v @ %v (%v)}",
		s.RouteId, s.ServiceId, s.StopId,
		common.SecsToTimeStr(s.DepartureSec), s.DepartureSec,
	)
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

	Scheduled []*Departure `json:"scheduled"`
	Live      []*Departure `json:"live"`
}

func (s *Stop) AppendLive() {
	calls, err := GetCallsByRouteStop(
		s.RouteId, strconv.Itoa(s.DirectionId),
		s.Id,
	)
	if err != nil {
		log.Println("can't append live schedules")
		return
	}

	sort.Sort(calls)
	for i := 0; i < len(calls) && i < maxStops; i++ {
		s.Live = append(s.Live, &Departure{
			Desc: calls[i].Extensions.Distances.PresentableDistance,
		})
	}
}

func (s Stop) String() string {
	return fmt.Sprintf("{%v %v %v %v @ (%v,%v)}",
		s.Id, s.Name, s.RouteId, s.Headsign, s.Lat, s.Lon,
	)
}

func (s Stop) Key() string {
	return fmt.Sprintf("%v%v", s.Id, s.RouteId)
}

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

func GetStopsByLoc(db sqlx.Ext, lat, lon, meters float64, filter string) (stops []*Stop, err error) {

	stops = []*Stop{}
	params := []interface{}{lat, lon, lat, lon, meters}

	// FIXME: this is not returning the closet stop
	q := `
		SELECT * FROM (
			SELECT
				DISTINCT ON (route_id, direction_id)
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

	q = q + ` 
			ORDER BY route_id, direction_id 
		) unique_routes

		ORDER BY dist ASC
	`

	err = sqlx.Select(db, &stops, q, params...)
	if err != nil {
		log.Println("can't get stop", err)
		return
	}

	now := time.Now()
	for _, stop := range stops {
		ydaysecs := []int64{}
		todaysecs := []int64{}

		allTimes := timeSlice{}

		yesterday := now.Add(-time.Hour * 12)
		yesterdayName := strings.ToLower(yesterday.Format("Monday"))
		todayName := strings.ToLower(now.Format("Monday"))

		if yesterdayName != todayName {
			var yesterdayId string
			// Looks for trips starting yesterday that arrive here
			// after midnight
			yesterdayId, err = getServiceIdByDay(
				db, stop.RouteId, yesterdayName, &now,
			)

			if err == sql.ErrNoRows {
				err = nil
				log.Println("no rows, ok, moving on")
				break
			}
			if err != nil {
				log.Println("can't get yesterday id", err)
				return
			}

			qYesterday := `
			SELECT scheduled_stop_time.departure_sec
			FROM   scheduled_stop_time
			WHERE  route_id   = $1 AND
			       stop_id    = $2 AND
				   service_id = $3 AND
				   departure_sec >= 86400 AND
				   departure_sec > $4
			ORDER BY departure_sec LIMIT $5
		`
			nowSecs :=
				now.Hour()*3600 + now.Minute()*60 + now.Second() + 86400

			err = sqlx.Select(db, &ydaysecs,
				qYesterday, stop.RouteId, stop.Id,
				yesterdayId, nowSecs, maxStops)

			if err != nil {
				log.Println("can't scan yesterday values", err)
				return
			}

			yesterday = yesterday.Add(
				-time.Hour * time.Duration(yesterday.Hour()))
			yesterday = yesterday.Add(
				-time.Minute * time.Duration(yesterday.Minute()))
			yesterday = yesterday.Add(
				-time.Second * time.Duration(yesterday.Second()))
			yesterday = yesterday.Add(
				-time.Nanosecond * time.Duration(yesterday.Nanosecond()))

			for _, ydaysec := range ydaysecs {
				thisTime := yesterday.Add(time.Second * time.Duration(ydaysec))
				allTimes = append(allTimes, thisTime)
			}
		}

		if true {
			var todayId string
			todayId, err = getServiceIdByDay(db, stop.RouteId, todayName, &now)
			if err == sql.ErrNoRows {
				err = nil
				log.Println("no rows there", err)
				break
			}
			if err != nil {
				log.Println("can't get today id", err)
				return
			}

			qToday := `
			SELECT scheduled_stop_time.departure_sec
			FROM   scheduled_stop_time
			WHERE  route_id   = $1 AND
			       stop_id    = $2 AND
				   service_id = $3 AND
				   departure_sec > $4
			ORDER BY departure_sec LIMIT $5
		`

			nowSecs := now.Hour()*3600 + now.Minute()*60 + now.Second()
			err = sqlx.Select(db, &todaysecs, qToday, stop.RouteId, stop.Id,
				todayId, nowSecs, maxStops)

			today := now
			today = today.Add(
				-time.Hour * time.Duration(today.Hour()))
			today = today.Add(
				-time.Minute * time.Duration(today.Minute()))
			today = today.Add(
				-time.Second * time.Duration(today.Second()))
			today = today.Add(
				-time.Nanosecond * time.Duration(today.Nanosecond()))

			for _, todaysec := range todaysecs {
				thisTime := today.Add(time.Second * time.Duration(todaysec))
				allTimes = append(allTimes, thisTime)
			}

			if err != nil {
				log.Println("can't scan today values", err)
				return
			}
		}

		sort.Sort(allTimes)

		for i, thisTime := range allTimes {
			if i > maxStops {
				break
			}
			stop.Scheduled = append(
				stop.Scheduled, &Departure{Time: thisTime},
			)
		}
		stop.AppendLive()
	}

	return stops, err
}

type ServiceRouteDay struct {
	ServiceId string
	RouteId   string
	Day       string

	StartDate time.Time
	EndDate   time.Time
}

func (s ServiceRouteDay) String() string {
	return fmt.Sprintf("{%v %v %v %v %v}",
		s.ServiceId, s.RouteId, s.Day, df(s.StartDate), df(s.EndDate),
	)
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

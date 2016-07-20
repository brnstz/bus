package models

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/brnstz/bus/internal/etc"
)

const (
	hereQueryLimit         = 10000
	departureLookaheadSecs = 60 * 60 * 3

	hereQuery = `
		SELECT
			agency_id,
			route_id,
			stop_id,
			service_id,
			trip_ids,
			arrival_secs,
			departure_secs,
			stop_sequences,

			stop_name,
			direction_id,
			stop_headsign,
			ST_X(location) AS lat,
			ST_Y(location) AS lon,

			route_type,
			route_color,
			route_text_color,

			trip_headsign,

			ST_DISTANCE(ST_GEOMFROMTEXT(:point_string, 4326), location) AS dist

		FROM here_trip

		WHERE
			ST_CONTAINS(ST_SETSRID(
				ST_MAKEPOLYGON(:line_string), 4326), location) AND

			( 
				service_id IN (%s) OR
				service_id IN (%s) OR
				service_id IN (%s) 
			)
	`

	routeTypeFilter = `
		AND route_type IN (%s)
	`

	hereOrderLimit = `
		ORDER BY dist ASC 
		LIMIT :limit
	`
)

type HereQuery struct {
	// The southwest and northeast bounding points of the box we are
	// searching
	SWLat float64 `db:"sw_lat"`
	SWLon float64 `db:"sw_lon"`
	NELat float64 `db:"ne_lat"`
	NELon float64 `db:"ne_lon"`

	// The midpoint of our search box
	MidLat float64 `db:"mid_lat"`
	MidLon float64 `db:"mid_lon"`

	LineString  string `db:"line_string"`
	PointString string `db:"point_string"`

	YesterdayDepartureMin  int
	YesterdayDepartureMax  int
	YesterdaySecDiff       int
	YesterdayDepartureBase time.Time
	YesterdayServiceIDs    []string
	YesterdayServiceIDMap  map[string]bool

	TodayDepartureMin  int
	TodayDepartureMax  int
	TodayDepartureBase time.Time
	TodayServiceIDs    []string
	TodayServiceIDMap  map[string]bool

	TomorrowDepartureMin  int
	TomorrowDepartureMax  int
	TomorrowSecDiff       int
	TomorrowDepartureBase time.Time
	TomorrowServiceIDs    []string
	TomorrowServiceIDMap  map[string]bool

	Limit int `db:"limit"`

	Query string
}

func NewHereQuery(lat, lon, swlat, swlon, nelat, nelon float64, routeTypes []int, now time.Time) (hq *HereQuery, err error) {

	// FIXME: hard coded, we need a region to agency mapping
	agencyID := "MTA NYCT"

	today := etc.BaseTime(now)
	todayName := strings.ToLower(now.Format("Monday"))
	todayMinSec := etc.TimeToDepartureSecs(now)
	todayMaxSec := todayMinSec + departureLookaheadSecs
	todayServiceIDs, err := GetNewServiceIDs(etc.DBConn, agencyID, todayName, today)
	if err != nil {
		log.Println("can't get today serviceIDs", err)
		return
	}

	yesterday := today.AddDate(0, 0, -1)
	yesterdayName := strings.ToLower(yesterday.Format("Monday"))
	yesterdayMinSec := now.Hour()*3600 + now.Minute()*60 + now.Second() + midnightSecs
	yesterdayMaxSec := yesterdayMinSec + departureLookaheadSecs
	yesterdaySecDiff := int(today.Sub(yesterday).Seconds())
	yesterdayServiceIDs, err := GetNewServiceIDs(etc.DBConn, agencyID, yesterdayName, yesterday)
	if err != nil {
		log.Println("can't get yesterday serviceIDs", err)
		return
	}

	tomorrow := today.AddDate(0, 0, 1)
	tomorrowName := strings.ToLower(tomorrow.Format("Monday"))
	tomorrowMinSec := 0
	tomorrowMaxSec := departureLookaheadSecs
	tomorrowSecDiff := int(tomorrow.Sub(today).Seconds())
	tomorrowServiceIDs, err := GetNewServiceIDs(etc.DBConn, agencyID, tomorrowName, tomorrow)
	if err != nil {
		log.Println("can't get tomorrow serviceIDs", err)
		return
	}

	// Check for overlap. If there is overlap, then nullify that day in
	// preference for today.
	if yesterdayMinSec <= todayMaxSec && todayMinSec <= yesterdayMaxSec {
		yesterdayMinSec = -1
		yesterdayMaxSec = -1
	}

	if tomorrowMinSec <= todayMaxSec && todayMinSec <= tomorrowMaxSec {
		tomorrowMinSec = -1
		tomorrowMaxSec = -1
	}

	// Check that yesterday is relevant
	if todayMinSec > departureLookaheadSecs {
		yesterdayMinSec = -1
		yesterdayMaxSec = -1
	}

	// Check that tomorrow is relevant
	if todayMaxSec < midnightSecs {
		tomorrowMinSec = -1
		tomorrowMaxSec = -1
	}

	hq = &HereQuery{
		MidLat: lat,
		MidLon: lon,
		SWLat:  swlat,
		SWLon:  swlon,
		NELat:  nelat,
		NELon:  nelon,
		Limit:  hereQueryLimit,

		TodayServiceIDs:    todayServiceIDs,
		TodayDepartureMin:  todayMinSec,
		TodayDepartureMax:  todayMaxSec,
		TodayDepartureBase: today,

		YesterdayServiceIDs:    yesterdayServiceIDs,
		YesterdayDepartureMin:  yesterdayMinSec,
		YesterdayDepartureMax:  yesterdayMaxSec,
		YesterdayDepartureBase: yesterday,
		YesterdaySecDiff:       yesterdaySecDiff,

		TomorrowServiceIDs:    tomorrowServiceIDs,
		TomorrowDepartureMin:  tomorrowMinSec,
		TomorrowDepartureMax:  tomorrowMaxSec,
		TomorrowDepartureBase: tomorrow,
		TomorrowSecDiff:       tomorrowSecDiff,
	}

	hq.LineString = fmt.Sprintf(
		`LINESTRING(%f %f, %f %f, %f %f, %f %f, %f %f)`,
		hq.SWLat, hq.SWLon,
		hq.SWLat, hq.NELon,
		hq.NELat, hq.NELon,
		hq.NELat, hq.SWLon,
		hq.SWLat, hq.SWLon,
	)

	hq.PointString = fmt.Sprintf(
		`POINT(%f %f)`,
		hq.MidLat, hq.MidLon,
	)

	hq.Query = fmt.Sprintf(hereQuery,
		etc.CreateIDs(hq.YesterdayServiceIDs),
		etc.CreateIDs(hq.TodayServiceIDs),
		etc.CreateIDs(hq.TomorrowServiceIDs),
	)

	if len(routeTypes) > 0 {
		hq.Query = hq.Query + fmt.Sprintf(routeTypeFilter, etc.CreateIntIDs(routeTypes))
	}

	hq.Query = hq.Query + hereOrderLimit

	hq.YesterdayServiceIDMap = map[string]bool{}
	for _, k := range hq.YesterdayServiceIDs {
		hq.YesterdayServiceIDMap[k] = true
	}

	hq.TodayServiceIDMap = map[string]bool{}
	for _, k := range hq.TodayServiceIDs {
		hq.TodayServiceIDMap[k] = true
	}

	hq.TomorrowServiceIDMap = map[string]bool{}
	for _, k := range hq.TomorrowServiceIDs {
		hq.TomorrowServiceIDMap[k] = true
	}

	return
}

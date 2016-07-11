package models

import (
	"log"
	"strings"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/jmoiron/sqlx"
)

func getNewServiceIDs(db sqlx.Ext, agencyID string, day string, now time.Time) (serviceIDs []string, err error) {

	var normalIDs []string
	var addedIDs []string
	var removedIDs []string

	if conf.API.LogTiming {
		t1 := time.Now()
		defer func() { log.Println(time.Now().Sub(t1)) }()
	}

	removed := map[string]bool{}

	// FIXME: do we want to select all service IDs that are valid
	// in this time window or just the max start date (most recent one)
	// Experience with MTA data suggests we only want one.

	// Select the service_id that:
	//   * matches our agencyID, day
	//   * has an end_date after now
	//   * has a start_date before now
	//   * has the maximum start date of those

	q := `
		SELECT service_id 
		FROM   service 
		WHERE  day = $1 AND
			   end_date >= $2 AND
			   start_date <= $3 AND 
			   agency_id = $4
		ORDER BY start_date DESC
		LIMIT 1
	`

	err = sqlx.Select(db, &normalIDs, q, day, now, now, agencyID)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, agencyID)
		return
	}

	// Get services added / removed
	q = `
		SELECT service_id 
		FROM   service_exception
		WHERE  exception_date = $1 AND
			   agency_id = $2 AND
			   exception_type = $3
	`

	// Added
	err = sqlx.Select(db, &addedIDs, q, now, agencyID, ServiceAdded)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, agencyID, ServiceAdded)
		return
	}

	// Removed
	err = sqlx.Select(db, &removedIDs, q, now, agencyID, ServiceRemoved)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, agencyID, ServiceRemoved)
		return
	}

	for _, v := range removedIDs {
		removed[v] = true
	}

	for _, v := range normalIDs {
		if !removed[v] {
			serviceIDs = append(serviceIDs, v)
		}
	}

	for _, v := range addedIDs {
		if !removed[v] {
			serviceIDs = append(serviceIDs, v)
		}
	}

	return

}

func GetStopsByHereQuery(db sqlx.Ext, hq HereQuery) (stops []*Stop, err error) {
	// overall function timing
	if conf.API.LogTiming {
		t1 := time.Now()
		defer func() { log.Println(time.Now().Sub(t1)) }()
	}

	now := time.Now()

	yesterday := baseTime(now.Add(-time.Hour * 12))
	today := baseTime(now)

	yesterdayName := strings.ToLower(yesterday.Format("Monday"))
	todayName := strings.ToLower(now.Format("Monday"))

	// FIXME: hard coded, we need a region to agency mapping
	agencyID := "MTA NYCT"

	todayIDs, err := getNewServiceIDs(db, agencyID, todayName, today)
	if err != nil {
		log.Println("can't get today IDs", err)
		return
	}
	yesterdayIDs, err := getNewServiceIDs(db, agencyID, yesterdayName, yesterday)
	if err != nil {
		log.Println("can't get yesterday IDs", err)
		return
	}

	hq.TodayServiceIDs = todayIDs
	hq.YesterdayServiceIDs = yesterdayIDs

	hq.YesterdayDepartureMin = now.Hour()*3600 + now.Minute()*60 + now.Second() + midnightSecs
	hq.YesterdayDepartureMax = hq.YesterdayDepartureMin + 60*60*3

	hq.TodayDepartureMin = now.Hour()*3600 + now.Minute()*60 + now.Second()
	hq.TodayDepartureMax = hq.TodayDepartureMin + 60*60*3

	err = hq.Initialize()
	if err != nil {
		log.Println("can't initialize hq", err)
		return
	}

	t3 := time.Now()
	rows, err := sqlx.NamedQuery(db, hq.Query, hq)
	if err != nil {
		log.Println("can't get stops", err)
		log.Printf("%s %+v", hq.Query, hq)
		return
	}
	if conf.API.LogTiming {
		log.Println(time.Now().Sub(t3))
	}

	defer rows.Close()

	log.Println("gettin rows", hq.Query, hq)
	for rows.Next() {
		log.Println("do i got a row?")
		var result map[string]interface{}

		err = rows.Scan(&result)
		if err != nil {
			log.Println("can't scan row")
		}

		log.Println(result)
	}

	return
}

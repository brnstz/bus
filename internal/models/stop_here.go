package models

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/jmoiron/sqlx"
)

type sortableStops []*Stop

func (ss sortableStops) Len() int {
	return len(ss)
}

func (ss sortableStops) Less(i, j int) bool {
	s1 := ss[i]
	s2 := ss[j]

	return s1.Dist < s2.Dist
}

func (ss sortableStops) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}

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

	// Previously we deduped this by route_id, but with bus / train data mixed
	// the agency_id is not unique enough to dedupe in same way. The main
	// problem is when we get updated files for services that haven't started yet
	// and these dates overlap with the previous service. We should probably
	// make the fix in the loader / materialized view.

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
	ss := sortableStops{}

	// mapping of stop.UniqueID to stop
	sm := map[string]*Stop{}

	// mapping of route.UniqueID + DirectionID to route
	rm := map[string]*Route{}

	// overall function timing
	if conf.API.LogTiming {
		t1 := time.Now()
		defer func() { log.Println(time.Now().Sub(t1)) }()
	}

	now := time.Now()

	//yesterday := baseTime(now.Add(-time.Hour * 12))
	today := baseTime(now)

	//yesterdayName := strings.ToLower(yesterday.Format("Monday"))
	todayName := strings.ToLower(now.Format("Monday"))

	// FIXME: hard coded, we need a region to agency mapping
	agencyID := "MTA NYCT"

	todayIDs, err := getNewServiceIDs(db, agencyID, todayName, today)
	if err != nil {
		log.Println("can't get today IDs", err)
		return
	}

	/* FIXME
	yesterdayIDs, err := getNewServiceIDs(db, agencyID, yesterdayName, yesterday)
	if err != nil {
		log.Println("can't get yesterday IDs", err)
		return
	}
	*/

	hq.ServiceIDs = todayIDs
	//hq.YesterdayServiceIDs = yesterdayIDs

	/*
		hq.YesterdayDepartureMin = now.Hour()*3600 + now.Minute()*60 + now.Second() + midnightSecs
		hq.YesterdayDepartureMax = hq.YesterdayDepartureMin + 60*60*3
	*/

	hq.DepartureMin = now.Hour()*3600 + now.Minute()*60 + now.Second()
	hq.DepartureMax = hq.DepartureMin + 60*60*3

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

	count := 0
	for rows.Next() {
		count++
		here := HereResult{DepartureBase: today}

		err = rows.StructScan(&here)
		if err != nil {
			log.Println("can't scan row", err)
			continue
		}

		err = here.Initialize()
		if err != nil {
			log.Println("can't initialize here", err)
			continue
		}

		routeDir := fmt.Sprintf("%v|%v", here.Route.UniqueID, here.Stop.DirectionID)

		oldStop, stopExists := sm[here.Stop.UniqueID]
		//oldRoute, routeExists := rm[routeDir]
		_, routeExists := rm[routeDir]

		// Ignore when the route / direction already exists, but stop is not
		// the same
		if routeExists && !stopExists {
			continue
		}

		// Ignore if it's our stop but we already have too many departures
		if stopExists && len(oldStop.Departures) >= MaxDepartures {
			continue
		}

		if !stopExists {
			sm[here.Stop.UniqueID] = here.Stop
		}
		if !routeExists {
			rm[routeDir] = here.Route
		}

		stop := sm[here.Stop.UniqueID]
		//route := rm[routeDir]

		stop.Departures = append(stop.Departures, here.Departure)
	}

	// Add all stops to sortableStops list
	for _, s := range sm {
		ss = append(ss, s)
	}

	// sort stops by distance
	sort.Sort(ss)

	stops = []*Stop(ss)

	for _, r := range rm {
		log.Println("what is the route?", r)
	}

	return
}

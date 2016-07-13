package models

import (
	"log"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/jmoiron/sqlx"
)

// Service is used by the Loader to map service IDs to route IDs
type Service struct {
	ID      string
	RouteID string
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

func getServiceIDsByDay(db sqlx.Ext, agencyID, routeID, day string, now time.Time) (serviceIDs []string, err error) {
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
	//   * matches our agencyID, routeID, day
	//   * has an end_date after now
	//   * has a start_date before now
	//   * has the maximum start date of those

	q := `
		SELECT service_id 
		FROM   service_route_day 
		WHERE  day = $1 AND
			   end_date >= $2 AND
			   start_date <= $3 AND 
			   route_id = $4 AND
			   agency_id = $5
		ORDER BY start_date DESC
		LIMIT 1
	`

	err = sqlx.Select(db, &normalIDs, q, day, now, now, routeID, agencyID)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, routeID, agencyID)
		return
	}

	// Get services added / removed
	q = `
		SELECT service_id 
		FROM   service_route_exception
		WHERE  exception_date = $1 AND
			   route_id = $2 AND
			   agency_id = $3 AND
			   exception_type = $4
	`

	// Added
	err = sqlx.Select(db, &addedIDs, q, now, routeID, agencyID, ServiceAdded)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, routeID, agencyID, ServiceAdded)
		return
	}

	// Removed
	err = sqlx.Select(db, &removedIDs, q, now, routeID, agencyID, ServiceRemoved)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, routeID, agencyID, ServiceRemoved)
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

package models

import (
	"fmt"
	"log"
	"time"

	"github.com/brnstz/bus/internal/etc"
	"github.com/jmoiron/sqlx"
)

// Service is used by the Loader to map service IDs to route IDs
type Service struct {
	ID      string
	RouteID string
}

func GetNewServiceIDs(db sqlx.Ext, agencyIDs []string, day string, now time.Time) (serviceIDs []string, err error) {

	var normalIDs []string
	var addedIDs []string
	var removedIDs []string

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

	q := fmt.Sprintf(`
		SELECT service_id 
		FROM   service 
		WHERE  day = $1 AND
			   end_date >= $2 AND
			   start_date <= $3 AND 
			   agency_id IN (%s)
	`, etc.CreateIDs(agencyIDs))

	err = sqlx.Select(db, &normalIDs, q, day, now, now)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, agencyIDs)
		return
	}

	// Get services added / removed
	q = fmt.Sprintf(`
		SELECT service_id 
		FROM   service_exception
		WHERE  exception_date = $1 AND
			   agency_id IN (%s) AND
			   exception_type = $2
	`, etc.CreateIDs(agencyIDs))

	// Added
	err = sqlx.Select(db, &addedIDs, q, now, ServiceAdded)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, agencyIDs, ServiceAdded)
		return
	}

	// Removed
	err = sqlx.Select(db, &removedIDs, q, now, ServiceRemoved)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, agencyIDs, ServiceRemoved)
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

func getServiceIDsByDay(db sqlx.Ext, agencyIDs []string, routeID, day string, now time.Time) (serviceIDs []string, err error) {
	var normalIDs []string
	var addedIDs []string
	var removedIDs []string

	removed := map[string]bool{}

	// FIXME: do we want to select all service IDs that are valid
	// in this time window or just the max start date (most recent one)
	// Experience with MTA data suggests we only want one.

	// Select the service_id that:
	//   * matches our agencyID, routeID, day
	//   * has an end_date after now
	//   * has a start_date before now
	//   * has the maximum start date of those

	q := fmt.Sprintf(`
		SELECT service_id 
		FROM   service_route_day 
		WHERE  day = $1 AND
			   end_date >= $2 AND
			   start_date <= $3 AND 
			   route_id = $4 AND
			   agency_id IN (%s)
		ORDER BY start_date DESC
		LIMIT 1
	`, etc.CreateIDs(agencyIDs))

	err = sqlx.Select(db, &normalIDs, q, day, now, now, routeID)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, routeID, agencyIDs)
		return
	}

	// Get services added / removed
	q = fmt.Sprintf(`
		SELECT service_id 
		FROM   service_route_exception
		WHERE  exception_date = $1 AND
			   route_id = $2 AND
			   agency_id IN (%s) AND
			   exception_type = $3
	`, etc.CreateIDs(agencyIDs))

	// Added
	err = sqlx.Select(db, &addedIDs, q, now, routeID, ServiceAdded)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, routeID, agencyIDs, ServiceAdded)
		return
	}

	// Removed
	err = sqlx.Select(db, &removedIDs, q, now, routeID, ServiceRemoved)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, routeID, agencyIDs, ServiceRemoved)
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

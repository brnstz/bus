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

// GetAgencyServiceIDs returns all possible serviceIDs for this day / time /
// agency for the initial query. However, these values may be later filtered
// by getRouteServiceIDs
func GetAgencyServiceIDs(db sqlx.Ext, agencyIDs []string, day string, now time.Time) (serviceIDs []string, err error) {

	var normalIDs []string
	var addedIDs []string
	var removedIDs []string

	inAgencyIDs := etc.CreateIDs(agencyIDs)

	removed := map[string]bool{}

	// Select all serviceIDs matching our agencies and within our time window
	q := fmt.Sprintf(`
		SELECT service_id 
		FROM   service 
		WHERE  day = $1 AND
			   end_date >= $2 AND
			   start_date <= $3 AND 
			   agency_id IN (%s)
	`, inAgencyIDs)

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
			   exception_type = $2 AND
			   agency_id IN (%s)
	`, inAgencyIDs)

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

	// Create mapping for removed IDs
	for _, v := range removedIDs {
		removed[v] = true
	}

	// Add all values from the service table that haven't been removed
	for _, v := range normalIDs {
		if !removed[v] {
			serviceIDs = append(serviceIDs, v)
		}
	}

	// Add all values from the service_execption table that haven't been
	// removed
	for _, v := range addedIDs {
		if !removed[v] {
			serviceIDs = append(serviceIDs, v)
		}
	}

	return
}

type routeService struct {
	AgencyID  string `db:"agency_id"`
	RouteID   string `db:"route_id"`
	ServiceID string `db:"service_id"`
}

func (rs *routeService) uniqueID() string {
	return rs.AgencyID + "|" + rs.RouteID
}

// getRouteServiceIDs returns the current relevant serviceIDs for these
// routes
func getRouteServiceIDs(db sqlx.Ext, agencyIDs, routeIDs []string, day string, now time.Time) (relevant map[string][]string, err error) {
	var rawNormalIDs []*routeService
	var normalIDs []*routeService
	var addedIDs []*routeService
	var removedIDs []*routeService
	var serviceIDs []*routeService

	inAgencyIDs := etc.CreateIDs(agencyIDs)
	inRouteIDs := etc.CreateIDs(routeIDs)

	removed := map[string]bool{}
	relevant = map[string][]string{}

	// Select all service

	q := fmt.Sprintf(`
		SELECT agency_id, route_id, service_id
		FROM   service_route_day 
		WHERE  day = $1 AND
			   end_date >= $2 AND
			   start_date <= $3 AND 
			   route_id IN (%s) AND
			   agency_id IN (%s)
		ORDER BY start_date DESC
	`, inRouteIDs, inAgencyIDs)

	err = sqlx.Select(db, &rawNormalIDs, q, day, now, now)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, agencyIDs)
		return
	}

	// Keep only the first value for each unique id
	var lastID string
	for _, v := range rawNormalIDs {
		id := v.uniqueID()
		// If it's the same as last time, then skip
		if id == lastID {
			continue
		}
		// Add to normal list
		normalIDs = append(normalIDs, v)

		// Set up for next iteration
		lastID = id
	}

	// Get services added / removed
	q = fmt.Sprintf(`
		SELECT agency_id, route_id, service_id 
		FROM   service_route_exception
		WHERE  exception_date = $1 AND
			   exception_type = $2 AND
			   route_id  IN (%s) AND
			   agency_id IN (%s) 
	`, inRouteIDs, inAgencyIDs)

	// Added
	err = sqlx.Select(db, &addedIDs, q, now, ServiceAdded)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, routeIDs, agencyIDs, ServiceAdded)
		return
	}

	// Removed
	err = sqlx.Select(db, &removedIDs, q, now, ServiceRemoved)
	if err != nil {
		log.Println("can't scan service ids", err, q, day, now, routeIDs, agencyIDs, ServiceRemoved)
		return
	}

	for _, v := range removedIDs {
		removed[v.uniqueID()] = true
	}

	for _, v := range normalIDs {
		if !removed[v.uniqueID()] {
			serviceIDs = append(serviceIDs, v)
		}
	}

	for _, v := range addedIDs {
		if !removed[v.uniqueID()] {
			serviceIDs = append(serviceIDs, v)
		}
	}

	for _, v := range serviceIDs {
		relevant[v.uniqueID()] = append(relevant[v.uniqueID()], v.ServiceID)
	}

	return
}

package models

import (
	"time"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
)

const (
	// Possible values for exception_type
	ServiceAdded   = 1
	ServiceRemoved = 2
)

type ServiceRouteException struct {
	AgencyID      string    `db:"agency_id" upsert:"key"`
	RouteID       string    `db:"route_id" upsert:"key"`
	ServiceID     string    `db:"service_id" upsert:"key"`
	ExceptionDate time.Time `db:"exception_date" upsert:"key"`
	ExceptionType int       `db:"exception_type"`
}

func (s *ServiceRouteException) Table() string {
	return "service_route_exception"
}

// Save saves a ServiceRouteDay to the database
func (s *ServiceRouteException) Save() error {
	_, err := upsert.Upsert(etc.DBConn, s)
	return err
}

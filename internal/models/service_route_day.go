package models

import (
	"fmt"
	"time"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
)

// ServiceRouteDay defines what service ID is valid for this route ID
// on this Day of the week. Service IDs are valid for a period of time
// which is indicated by StartDate and EndDate. All fields in this
// object form a unique key.
type ServiceRouteDay struct {
	ServiceID string `db:"service_id" upsert:"key"`
	RouteID   string `db:"route_id" upsert:"key"`
	Day       string `db:"day" upsert:"key"`

	StartDate time.Time `db:"start_date" upsert:"key"`
	EndDate   time.Time `db:"end_date" upsert:"key"`
}

// Table implements the upsert.Upserter interface by defining the
// table we're saving to
func (s *ServiceRouteDay) Table() string {
	return "service_route_day"
}

// Save saves a ServiceRouteDay to the database
func (s *ServiceRouteDay) Save() error {
	_, err := upsert.Upsert(etc.DBConn, s)
	return err
}

// String returns a text representation of the ServiceRouteDay
func (s ServiceRouteDay) String() string {
	return fmt.Sprintf("{%v %v %v %v %v}",
		s.ServiceID, s.RouteID, s.Day, df(s.StartDate), df(s.EndDate),
	)
}

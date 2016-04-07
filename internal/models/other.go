package models

import "time"

// This file contains transient types that aren't stored in the db

type Trip struct {
	ID          string
	Headsign    string
	DirectionID int
}

type ServiceRouteException struct {
	ServiceID     string
	RouteID       string
	ExceptionDate time.Time
}

type Departure struct {
	Time time.Time `json:"time" db:"time"`
	Desc string    `json:"desc"`

	// FIXME: stops away? miles away?
}

type Service struct {
	ID      string
	RouteID string
}

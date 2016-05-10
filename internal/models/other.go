package models

import "time"

// This file contains transient types that aren't stored in the db

type ServiceRouteException struct {
	ServiceID     string
	RouteID       string
	ExceptionDate time.Time
}

type Departure struct {
	Time time.Time `json:"time" db:"time"`
}

type Service struct {
	ID      string
	RouteID string
}

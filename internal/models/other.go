package models

import "time"

// This file contains transient types that aren't stored in the db

type Service struct {
	ID      string
	RouteID string
}

// baseTime takes a time and returns the same time with the hour, minute, second
// and nanosecond values set so zero, so that it represents the start
// of the day
func baseTime(t time.Time) time.Time {
	t = t.Add(-time.Hour * time.Duration(t.Hour()))
	t = t.Add(-time.Minute * time.Duration(t.Minute()))
	t = t.Add(-time.Second * time.Duration(t.Second()))
	t = t.Add(-time.Nanosecond * time.Duration(t.Nanosecond()))

	return t
}

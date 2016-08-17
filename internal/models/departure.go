package models

import "time"

const (
	// midnightSecs is what a depature_sec value of midnight looks like. We
	// use this for trying to find departures after midnight.
	midnightSecs = 86400

	// maximum departures to return in a result
	MaxDepartures = 6
)

type Departure struct {
	Time         time.Time `json:"time" db:"-"`
	DepartureSec int       `json:"-" db:"departure_sec"`
	TripID       string    `json:"trip_id" db:"trip_id"`
	ServiceID    string    `json:"service_id" db:"service_id"`
	Live         bool      `json:"live" db:"-" upsert:"omit"`

	// CompassDir is the direction to the next stop
	CompassDir float64 `json:"compass_dir" db:"-" upsert:"omit"`

	baseTime time.Time `json:"-" db:"-"`
}

func (d *Departure) Initialize() error {
	d.Time = d.baseTime.Add(time.Second * time.Duration(d.DepartureSec))

	return nil
}

type SortableDepartures []*Departure

func (d SortableDepartures) Len() int {
	return len(d)
}

func (d SortableDepartures) Less(i, j int) bool {
	return d[i].Time.Before(d[j].Time)
}

func (d SortableDepartures) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

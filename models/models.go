package models

import "fmt"

type Stop struct {
	Id          string
	Name        string
	RouteId     string
	StationType string

	DirectionId int
	Headsign    string

	Lat float64
	Lon float64
}

func (s Stop) String() string {
	return fmt.Sprintf("{%v %v %v @ (%v,%v)}", s.Id, s.Name, s.RouteId, s.Lat, s.Lon)
}

func (s Stop) Key() string {
	return fmt.Sprintf("%v%v", s.Id, s.RouteId)
}

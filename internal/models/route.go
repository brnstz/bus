package models

import (
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
	"github.com/jmoiron/sqlx"
)

const (
	Tram int = iota
	Subway
	Rail
	Bus
	Ferry
	CableCar
	Gondola
	Funicular
)

const (
	defaultColor     = "FFFFFF"
	defaultTextColor = "000000"
)

var (
	// routeTypeString maps route_id codes to strings
	routeTypeString = map[int]string{
		Tram:      "tram",
		Subway:    "subway",
		Rail:      "rail",
		Bus:       "bus",
		Ferry:     "ferry",
		CableCar:  "cable_car",
		Gondola:   "gondola",
		Funicular: "funicular",
	}

	// routeTypeInt maps route_id strings to int codes
	routeTypeInt = map[string]int{}
)

func init() {
	// initialize routeTypeInt using the reverse mapping in routeTypeString
	for k, v := range routeTypeString {
		routeTypeInt[v] = k
	}
}

// Route is https://developers.google.com/transit/gtfs/reference#routestxt
type Route struct {
	ID        string `json:"route_id" db:"route_id" upsert:"key"`
	Type      int    `json:"route_type" db:"route_type"`
	TypeName  string `json:"route_type_name" db:"-" upsert:"omit"`
	Color     string `json:"route_color" db:"route_color"`
	TextColor string `json:"route_text_color" db:"route_text_color"`
}

// Table returns the table name for the Route struct, implementing the
// upsert.Upserter interface
func (r *Route) Table() string {
	return "route"
}

// NewRoute creates a Route given incoming data, typically from a routes.txt
// file
func NewRoute(id string, rtype int, color, textColor string) (r *Route, err error) {
	var ok bool

	if len(color) < 1 {
		color = defaultColor
	}

	if len(textColor) < 1 {
		textColor = defaultTextColor
	}

	if len(id) < 1 {
		err = ErrNoID
		return
	}

	r = &Route{
		ID:        id,
		Type:      rtype,
		Color:     color,
		TextColor: textColor,
	}

	// Load the string name of the route type, also checking that the incoming
	// rtype was correct
	r.TypeName, ok = routeTypeString[r.Type]
	if !ok {
		r = nil
		err = ErrInvalidRouteType
		return
	}

	return
}

// GetRoute returns a Route with the given ID
func GetRoute(id string) (r *Route, err error) {
	var ok bool

	r = &Route{}
	err = sqlx.Get(etc.DBConn, r, `SELECT * FROM route WHERE route_id = $1`, id)
	if err != nil {
		return
	}

	// Load the string name of the route type
	r.TypeName, ok = routeTypeString[r.Type]
	if !ok {
		r = nil
		err = ErrInvalidRouteType
		return
	}

	return
}

// Save saves a route to the database
func (r *Route) Save() error {
	_, err := upsert.Upsert(etc.DBConn, r)
	return err
}

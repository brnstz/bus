package models

import (
	"log"
	"strings"

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
	defaultColor     = "#FFFFFF"
	defaultTextColor = "#000000"
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
	RouteID   string `json:"route_id" db:"route_id" upsert:"key"`
	AgencyID  string `json:"agency_id" db:"agency_id" upsert:"key"`
	Type      int    `json:"route_type" db:"route_type"`
	TypeName  string `json:"route_type_name" db:"-" upsert:"omit"`
	Color     string `json:"route_color" db:"route_color"`
	TextColor string `json:"route_text_color" db:"route_text_color"`

	UniqueID    string        `json:"unique_id" db:"-" upsert:"omit"`
	RouteShapes []*RouteShape `json:"route_shapes" upsert:"omit"`
	Stops       []*Stop       `json:"stops" upsert:"omit"`
}

// Table returns the table name for the Route struct, implementing the
// upsert.Upserter interface
func (r *Route) Table() string {
	return "route"
}

// init ensures any derived values are correct after creating/loading
// an object
func (r *Route) init() (err error) {
	var ok bool

	// Load the string name of the route type, also checking that the incoming
	// rtype was correct
	r.TypeName, ok = routeTypeString[r.Type]
	if !ok {
		r = nil
		err = ErrInvalidRouteType
		return
	}

	r.UniqueID = r.AgencyID + "|" + r.RouteID

	return
}

// checkColor ensures that color is a non-empty string and ensures
// it is prepended by #
func checkColor(color, def string) string {
	color = strings.TrimSpace(color)
	if len(color) < 1 {
		color = def
	}

	if color[0] != '#' {
		color = "#" + color
	}

	return color
}

// NewRoute creates a Route given incoming data, typically from a routes.txt
// file
func NewRoute(id string, rtype int, color, textColor, agencyID string) (r *Route, err error) {

	color = checkColor(color, defaultColor)
	textColor = checkColor(textColor, defaultTextColor)

	if len(id) < 1 {
		err = ErrNoID
		return
	}

	r = &Route{
		RouteID:   id,
		Type:      rtype,
		Color:     color,
		TextColor: textColor,
		AgencyID:  agencyID,
	}

	err = r.init()
	if err != nil {
		log.Println(err)
		return
	}

	return
}

// test func for static json file
func GetAllRoutes(db sqlx.Ext, agencyID string) (routes []*Route, err error) {
	err = sqlx.Select(db, &routes,
		`SELECT * FROM route WHERE agency_id = $1 AND route_type != $2`,
		agencyID, Bus,
	)
	if err != nil {
		return
	}

	for _, r := range routes {
		err = r.init()
		if err != nil {
			return
		}

		r.RouteShapes, err = GetSavedRouteShapes(
			etc.DBConn, r.AgencyID, r.RouteID,
		)
		if err != nil {
			log.Println("can't append shapes", err)
			return
		}

	}

	return

}

// GetRoute returns a Route with the given ID
func GetRoute(db sqlx.Ext, agencyID, routeID string) (r *Route, err error) {

	r = &Route{}
	err = sqlx.Get(db, r,
		`SELECT * FROM route WHERE agency_id = $1 AND route_id = $2`,
		agencyID, routeID,
	)
	if err != nil {
		return
	}

	err = r.init()
	if err != nil {
		return
	}

	return
}

// Save saves a route to the database
func (r *Route) Save() error {
	_, err := upsert.Upsert(etc.DBConn, r)
	return err
}

package models

import (
	"fmt"
	"log"
	"strconv"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/upsert"
	"github.com/jmoiron/sqlx"
)

// Stop is a single transit stop for a particular route. If a
// stop serves more than one route, there are multiple distinct
// entries for that stop.
type Stop struct {
	StopID      string `json:"stop_id" db:"stop_id" upsert:"key"`
	RouteID     string `json:"route_id" db:"route_id" upsert:"key"`
	AgencyID    string `json:"agency_id" db:"agency_id" upsert:"key"`
	DirectionID int    `json:"direction_id" db:"direction_id" upsert:"key"`
	Name        string `json:"stop_name" db:"stop_name"`

	UniqueID string `json:"unique_id" db:"-" upsert:"omit"`

	Headsign string `json:"headsign" db:"headsign"`

	Lat float64 `json:"lat" db:"lat" upsert:"omit"`
	Lon float64 `json:"lon" db:"lon" upsert:"omit"`

	// Location is PostGIS field value that combines lat and lon into a single
	// field.
	Location interface{} `json:"-" db:"location" upsert_value:"ST_SetSRID(ST_MakePoint(:lat, :lon),4326)"`

	// info we steal from route when doing a here query
	RouteType      int    `json:"route_type" db:"-" upsert:"omit"`
	RouteTypeName  string `json:"route_type_name" db:"-" upsert:"omit"`
	RouteColor     string `json:"route_color" db:"-" upsert:"omit"`
	RouteTextColor string `json:"route_text_color" db:"-" upsert:"omit"`

	Seq int `json:"seq" db:"stop_sequence" upsert:"omit"`

	Dist       float64      `json:"dist,omitempty" db:"-" upsert:"omit"`
	Departures []*Departure `json:"departures,omitempty" db:"-" upsert:"omit"`
	Vehicles   []Vehicle    `json:"vehicles,omitempty" db:"-" upsert:"omit"`
}

func (s *Stop) Initialize() error {
	s.UniqueID = s.AgencyID + "|" + s.RouteID + "|" + s.StopID + "|" + strconv.Itoa(s.DirectionID)

	// If there is a route type defined, then load its name. Ignore errors.
	s.RouteTypeName = routeTypeString[s.RouteType]

	return nil
}

// Table implements the upsert.Upserter interface, returning the table
// where we save stops.
func (s *Stop) Table() string {
	return "stop"
}

// Save saves a stop to the database
func (s *Stop) Save() error {
	_, err := upsert.Upsert(etc.DBConn, s)
	return err
}

// String returns a descriptive string for this stop.
func (s Stop) String() string {
	return fmt.Sprintf("{%v %v %v %v %v @ (%v,%v)}",
		s.StopID, s.Name, s.RouteID, s.Headsign, s.DirectionID, s.Lat, s.Lon,
	)
}

// Key() returns the unique string for this stop, so we can identify
// unique stops in the loader.
func (s Stop) Key() string {
	return fmt.Sprintf("%v%v", s.StopID, s.RouteID)
}

func GetStopsByTrip(db sqlx.Ext, t *Trip) (stops []*Stop, err error) {

	q := `
		SELECT stop.*, 
			ST_X(location) AS lat, 
			ST_Y(location) AS lon,
			sst.stop_sequence

		FROM stop
		INNER JOIN scheduled_stop_time sst 
			ON stop.agency_id = sst.agency_id AND
			   stop.route_id  = sst.route_id  AND
			   stop.stop_id   = sst.stop_id

		WHERE sst.agency_id     = $1 AND
			  sst.route_id      = $2 AND
	          sst.trip_id       = $3 AND
			  stop.direction_id = $4

		ORDER by sst.stop_sequence ASC
	`

	err = sqlx.Select(db, &stops, q,
		t.AgencyID, t.RouteID, t.TripID, t.DirectionID,
	)

	if err != nil {
		log.Println("can't get trips", err)
		return
	}

	return
}

// we want to sort stops first by their type, then by dist (i.e.,
// show subways before buses even if bus is closer)
const (
	byDist  = 0
	byType  = 1
	byRoute = 2
	byDir   = 3
)

type sortableStops struct {
	stops []*Stop

	// map of agency_id|route_id to distance
	maxRouteDist map[string]float64

	by int
}

func (ss sortableStops) Len() int {
	return len(ss.stops)
}

func (ss sortableStops) distID(s *Stop) string {
	return s.AgencyID + "|" + s.RouteID
}

func (ss sortableStops) Less(i, j int) bool {

	switch ss.by {

	case byDist:
		d1 := ss.maxRouteDist[ss.distID(ss.stops[i])]
		d2 := ss.maxRouteDist[ss.distID(ss.stops[j])]

		return d1 < d2

	case byType:
		return routeTypeSort[ss.stops[i].RouteType] < routeTypeSort[ss.stops[j].RouteType]

	case byRoute:
		return ss.stops[i].RouteID < ss.stops[j].RouteID

	case byDir:
		return ss.stops[i].DirectionID < ss.stops[j].DirectionID
	}

	log.Println("unrecognized sort by", ss.by)
	return false
}

func (ss sortableStops) Swap(i, j int) {
	ss.stops[i], ss.stops[j] = ss.stops[j], ss.stops[i]
}

func newSortableStops(stops []*Stop) (ss sortableStops) {
	ss = sortableStops{
		stops:        stops,
		maxRouteDist: map[string]float64{},
	}

	for _, s := range stops {
		dist := s.Dist
		id := ss.distID(s)

		if dist > ss.maxRouteDist[id] {
			ss.maxRouteDist[id] = dist
		}
	}

	return
}

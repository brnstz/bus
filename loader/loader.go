package loader

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/brnstz/bus/models"
)

type Loader struct {
	// the dir from which we load google transit files
	dir string

	// mapping from stop_id to a slice of trips
	stopTrips map[string][]string

	// mapping trip_id to route_id
	tripRoute map[string]string

	// a map of "{stop_id}-{route_id}" to stop objects. Essentially a
	// list of unique stops by route.
	uniqueStop map[string]*models.Stop

	Stops []*models.Stop
}

func NewLoader(dir string) *Loader {
	l := Loader{
		dir:        dir,
		stopTrips:  map[string][]string{},
		tripRoute:  map[string]string{},
		uniqueStop: map[string]*models.Stop{},
	}

	l.init()

	return &l
}

func (l *Loader) init() {
	l.loadStopTrips()
	l.loadTripRoute()
	l.loadUniqueStop()

	l.Stops = make([]*models.Stop, len(l.uniqueStop))

	i := 0
	for _, v := range l.uniqueStop {
		l.Stops[i] = v
		i++
	}
}

func getcsv(dir, name string) *csv.Reader {

	f, err := os.Open(path.Join(dir, name))
	if err != nil {
		panic(err)
	}

	return csv.NewReader(f)
}

// find index of col in header

func find(header []string, col string) int {
	for i := 0; i < len(header); i++ {
		if header[i] == col {
			return i
		}
	}

	log.Fatalf("can't find header col %v", col)
	return -1
}

func (l *Loader) loadStopTrips() {
	stop_times := getcsv(l.dir, "stop_times.txt")

	header, err := stop_times.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	stop_index := find(header, "stop_id")
	trip_index := find(header, "trip_id")
	for i := 0; ; i++ {
		rec, err := stop_times.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
		}

		stop := rec[stop_index]
		trip := rec[trip_index]

		l.stopTrips[stop] = append(l.stopTrips[stop], trip)
	}
}

func (l *Loader) loadTripRoute() {

	trips := getcsv(l.dir, "trips.txt")

	header, err := trips.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	trip_index := find(header, "trip_id")
	route_index := find(header, "route_id")

	trips.Read()
	for i := 0; ; i++ {
		rec, err := trips.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("%v on line %v of trips.txt", err, i)
		}

		trip := rec[trip_index]
		route := rec[route_index]

		l.tripRoute[trip] = route
	}
}

func (l *Loader) loadUniqueStop() {
	stops := getcsv(l.dir, "stops.txt")

	header, err := stops.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	stop_index := find(header, "stop_id")
	stop_name_index := find(header, "stop_name")
	stop_lat_index := find(header, "stop_lat")
	stop_lon_index := find(header, "stop_lon")

	for i := 0; ; i++ {
		rec, err := stops.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("%v on line %v of stops.txt", err, i)
		}

		stop := rec[stop_index]
		stopName := rec[stop_name_index]

		stopLat, err := strconv.ParseFloat(strings.TrimSpace(rec[stop_lat_index]), 64)
		if err != nil {
			log.Fatalf("%v on line %v of stops.txt", err, i)
		}

		stopLon, err := strconv.ParseFloat(strings.TrimSpace(rec[stop_lon_index]), 64)
		if err != nil {
			log.Fatalf("%v on line %v of stops.txt", err, i)
		}

		trips, exists := l.stopTrips[stop]
		if exists {
			for _, trip := range trips {
				obj := models.Stop{
					Id:      stop,
					Name:    stopName,
					Lat:     stopLat,
					Lon:     stopLon,
					RouteId: l.tripRoute[trip],
				}
				l.uniqueStop[obj.Key()] = &obj
			}
		}
	}
}

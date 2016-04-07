package loader

import (
	"encoding/csv"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
)

var days = []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}

var datefmt = "20060102"

type Loader struct {
	// the dir from which we load google transit files
	dir string

	// mapping from trip id to a trip object
	trips map[string]*models.Trip

	// mapping from stop_id to a slice of trip_ids
	stopTrips map[string][]string

	// mapping trip_id to route_id
	tripRoute map[string]string

	// a map of "{stop_id}-{route_id}" to stop objects. Essentially a
	// list of unique stops by route.
	uniqueStop map[string]*models.Stop

	// mapping of trip_id to service object
	tripService map[string]*models.Service

	// mapping of service_id to map of unique route_id
	serviceRoute map[string]map[string]bool

	routeIDs map[string]bool

	Routes []*models.Route

	Stops []*models.Stop

	ScheduledStopTimes []*models.ScheduledStopTime

	ServiceRouteDays       []*models.ServiceRouteDay
	ServiceRouteExceptions []*models.ServiceRouteException
}

func NewLoader(dir string, routeFilters []string) *Loader {
	l := Loader{
		dir:          dir,
		trips:        map[string]*models.Trip{},
		stopTrips:    map[string][]string{},
		tripRoute:    map[string]string{},
		uniqueStop:   map[string]*models.Stop{},
		tripService:  map[string]*models.Service{},
		serviceRoute: map[string]map[string]bool{},
	}

	if len(routeFilters) > 0 {
		l.routeIDs = map[string]bool{}
		for _, v := range routeFilters {
			l.routeIDs[v] = true
		}
	}

	l.init()

	return &l
}

func (l *Loader) init() {
	l.loadRoutes()
	l.loadTrips()
	l.loadStopTrips()
	l.loadTripRoute()
	l.loadUniqueStop()
	l.loadCalendars()

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

	r := csv.NewReader(f)
	r.LazyQuotes = true
	return r
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

// skipRoute returns true if we should skip this route given our routeFilter
// config
func (l *Loader) skipRoute(routeID string) bool {
	if l.routeIDs != nil && l.routeIDs[routeID] == false {
		return true
	} else {
		return false
	}
}

func (l *Loader) loadRoutes() {
	f := getcsv(l.dir, "routes.txt")

	header, err := f.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	routeIdx := find(header, "route_id")
	routeTypeIdx := find(header, "route_type")
	routeColorIdx := find(header, "route_color")
	routeTextColorIdx := find(header, "route_text_color")

	for i := 0; ; i++ {
		rec, err := f.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v on line %v of routes.txt", err, i)
		}

		route := rec[routeIdx]
		if l.skipRoute(route) {
			continue
		}

		routeType, err := strconv.Atoi(rec[routeTypeIdx])
		if err != nil {
			log.Fatalf("%v on line %v of routes.txt", err, i)
		}

		routeColor := rec[routeColorIdx]
		routeTextColor := rec[routeTextColorIdx]

		r, err := models.NewRoute(
			route, routeType, routeColor, routeTextColor,
		)
		if err != nil {
			log.Fatalf("%v on line %v of routes.txt", err, i)
		}

		l.Routes = append(l.Routes, r)
	}

}

func (l *Loader) loadTrips() {
	f := getcsv(l.dir, "trips.txt")

	header, err := f.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	tripIdx := find(header, "trip_id")
	dirIdx := find(header, "direction_id")
	headIdx := find(header, "trip_headsign")
	serviceIdx := find(header, "service_id")
	routeIdx := find(header, "route_id")

	for i := 0; ; i++ {
		rec, err := f.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("%v on line %v of trips.txt", err, i)
		}
		direction, err := strconv.Atoi(rec[dirIdx])
		if err != nil {
			log.Fatalf("%v on line %v of trips.txt", err, i)
		}

		trip := &models.Trip{
			Id:          rec[tripIdx],
			DirectionId: direction,
			Headsign:    rec[headIdx],
		}

		service := rec[serviceIdx]
		route := rec[routeIdx]

		if l.skipRoute(route) {
			continue
		}

		l.trips[trip.Id] = trip

		serviceObj := &models.Service{
			Id:      service,
			RouteId: route,
		}

		l.tripService[trip.Id] = serviceObj

		if l.serviceRoute[service] == nil {
			l.serviceRoute[service] = map[string]bool{}
		}
		l.serviceRoute[service][route] = true
	}

}

func (l *Loader) loadStopTrips() {
	stopTimes := getcsv(l.dir, "stop_times.txt")

	header, err := stopTimes.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	stopIdx := find(header, "stop_id")
	tripIdx := find(header, "trip_id")
	timeIdx := find(header, "departure_time")
	for i := 0; ; i++ {
		rec, err := stopTimes.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("%v on line %v of stop_times.txt", err, i)
		}

		stop := rec[stopIdx]
		trip := rec[tripIdx]
		timeStr := rec[timeIdx]

		l.stopTrips[stop] = append(l.stopTrips[stop], trip)

		service, exists := l.tripService[trip]
		if !exists {
			continue
		}

		sst, err := models.NewScheduledStopTime(service.RouteId, stop, service.Id, timeStr)
		if err != nil {
			log.Fatal("can't create sst", rec, err)
		}

		l.ScheduledStopTimes = append(l.ScheduledStopTimes, &sst)

	}
}

func (l *Loader) loadTripRoute() {

	trips := getcsv(l.dir, "trips.txt")

	header, err := trips.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	tripIdx := find(header, "trip_id")
	routeIdx := find(header, "route_id")

	trips.Read()
	for i := 0; ; i++ {
		rec, err := trips.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("%v on line %v of trips.txt", err, i)
		}

		trip := rec[tripIdx]
		route := rec[routeIdx]

		if l.skipRoute(route) {
			continue
		}

		l.tripRoute[trip] = route
	}
}

func (l *Loader) loadUniqueStop() {
	stops := getcsv(l.dir, "stops.txt")

	header, err := stops.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	stopIdx := find(header, "stop_id")
	stopNameIdx := find(header, "stop_name")
	stopLatIdx := find(header, "stop_lat")
	stopLonIdx := find(header, "stop_lon")

	for i := 0; ; i++ {
		rec, err := stops.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("%v on line %v of stops.txt", err, i)
		}

		stopLat, err := strconv.ParseFloat(
			strings.TrimSpace(rec[stopLatIdx]), 64,
		)
		if err != nil {
			log.Fatalf("%v on line %v of stops.txt", err, i)
		}

		stopLon, err := strconv.ParseFloat(
			strings.TrimSpace(rec[stopLonIdx]), 64,
		)
		if err != nil {
			log.Fatalf("%v on line %v of stops.txt", err, i)
		}

		trips, exists := l.stopTrips[rec[stopIdx]]
		if exists {
			for _, trip := range trips {
				if l.skipRoute(l.tripRoute[trip]) {
					continue
				}
				obj := models.Stop{
					Id:      rec[stopIdx],
					Name:    rec[stopNameIdx],
					Lat:     stopLat,
					Lon:     stopLon,
					RouteId: l.tripRoute[trip],

					DirectionId: l.trips[trip].DirectionId,
					Headsign:    l.trips[trip].Headsign,
				}
				l.uniqueStop[obj.Key()] = &obj
			}
		}
	}
}

func (l *Loader) loadCalendars() {
	stops := getcsv(l.dir, "calendar.txt")

	header, err := stops.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	idxs := map[string]int{}
	for _, day := range days {
		idxs[day] = find(header, day)
	}
	serviceIdx := find(header, "service_id")
	startDateIdx := find(header, "start_date")
	endDateIdx := find(header, "end_date")

	for i := 0; ; i++ {
		rec, err := stops.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("%v on line %v of calendar.txt", err, i)
		}

		serviceId := rec[serviceIdx]

		startDate, err := time.Parse(datefmt, rec[startDateIdx])
		if err != nil {
			log.Fatalf("can't parse start date %v %v", err, rec[startDateIdx])
		}

		endDate, err := time.Parse(datefmt, rec[endDateIdx])
		if err != nil {
			log.Fatalf("can't parse end date %v %v", err, rec[endDateIdx])
		}

		for day, dayIdx := range idxs {
			dayVal := rec[dayIdx]
			if dayVal != "1" {
				continue
			}
			for route, _ := range l.serviceRoute[serviceId] {
				srd := models.ServiceRouteDay{
					ServiceID: serviceId,
					RouteID:   route,
					Day:       day,
					StartDate: startDate,
					EndDate:   endDate,
				}

				l.ServiceRouteDays = append(l.ServiceRouteDays, &srd)
			}
		}
	}
}

func doOne(dir string, routeFilters []string) {
	var err error

	db := etc.DBConn

	l := NewLoader(dir, routeFilters)

	for _, r := range l.Routes {
		err = r.Save()
		if err != nil {
			log.Fatalf("cannot save route: %v", err)
		}
	}

	for i, s := range l.ServiceRouteDays {
		err = s.Save()
		if err != nil {
			log.Fatalf("cannot save route: %v", err)
		}

		if i%100 == 0 && i > 0 {
			log.Printf("loaded %v service route days", i)
		}

	}

	for i, s := range l.Stops {
		_, err := db.Exec(`
				INSERT INTO stop
				(stop_id, stop_name, direction_id, headsign, route_id,
				 location)
				VALUES($1, $2, $3, $4, $5, ll_to_earth($6, $7))
			`,
			s.Id, s.Name, s.DirectionId, s.Headsign, s.RouteId,
			s.Lat, s.Lon,
		)

		if err != nil && !strings.Contains(err.Error(), "violates unique constraint") {
			log.Println("ERROR STOPS: ", err, s)
		}

		if i%100 == 0 && i > 0 {
			log.Printf("loaded %v stops", i)
		}
	}

	for i, s := range l.ScheduledStopTimes {
		_, err := db.Exec(`
				INSERT INTO scheduled_stop_time
				(route_id, stop_id, service_id, departure_sec)
				VALUES($1, $2, $3, $4)
			`, s.RouteId, s.StopId, s.ServiceId, s.DepartureSec,
		)
		if err != nil && !strings.Contains(err.Error(), "violates unique constraint") {
			log.Println("ERROR SCHEDULED STOP TIMES: ", err, s)
		}

		if i%100000 == 0 && i > 0 {
			log.Printf("loaded %v stop times", i)
		}
	}
}

func LoadOnce(routeFilters []string, urls ...string) {
	for _, url := range urls {

		// FIXME: do this in Go, need to make it integrated with loader
		dir, err := ioutil.TempDir(conf.Loader.TmpDir, "")
		if err != nil {
			panic(err)
		}
		cmd := exec.Command("wget", url, "-O", path.Join(dir, "file.zip"))
		err = cmd.Run()
		if err != nil {
			panic(err)
		}

		cmd = exec.Command("unzip", path.Join(dir, "file.zip"), "-d", dir)
		err = cmd.Run()
		if err != nil {
			panic(err)
		}

		func() {
			log.Println(url, dir)
			defer os.RemoveAll(dir)
			t1 := time.Now()
			doOne(dir, routeFilters)
			t2 := time.Now()
			log.Printf("took %v for %v\n", t2.Sub(t1), dir)
		}()

	}
}

func LoadForever(routeFilters []string, urls ...string) {
	for {
		LoadOnce(routeFilters, urls...)
		log.Println("finished loading, sleeping for 24 hours")
		time.Sleep(time.Hour * 24)
	}
}

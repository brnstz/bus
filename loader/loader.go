package loader

import (
	"fmt"
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

var (
	days        = []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	datefmt     = "20060102"
	loaderBreak = time.Hour * 24

	views = []string{"here_trip", "service", "service_exception"}

	logp = 1000
)

// rskey is the unique key for a route_shape
type rskey struct {
	routeID     string
	directionID int
	headsign    string
}

type latlon struct {
	lat float64
	lon float64
}

type Loader struct {
	// the dir from which we load google transit files
	dir string

	// mapping from trip id to a trip object
	trips map[string]*models.Trip

	// mapping from stop_id to a slice of trip_ids
	stopTrips map[string][]string

	// mapping of trip_id to service object
	tripService map[string]*models.Service

	// mapping of service_id to map of unique route_id
	serviceRoute map[string]map[string]bool

	routeIDs map[string]bool

	// routeAgency contains the agency for each route after we loadRoutes()
	routeAgency map[string]string

	// mapping trip_id to route_id
	tripRoute map[string]string

	// shapeRoute maps shape_id to route_id (for purposes of adding agency_id
	// to shapes table)
	shapeRoute map[string]string

	// maxTripSeq
	maxTripSeq map[string]int

	// mapping of stop ids to lat/lon pairs
	stopLocation map[string]*latlon

	// routeShapeCount keeps a running tab of the biggest shape for this
	// route/dir/headsign combo
	/*
		routeShapeCount map[rskey]int
		routeShapeID map[rskey]
	*/
}

func newLoader(dir string) *Loader {
	l := Loader{
		dir:          dir,
		trips:        map[string]*models.Trip{},
		stopTrips:    map[string][]string{},
		tripRoute:    map[string]string{},
		tripService:  map[string]*models.Service{},
		serviceRoute: map[string]map[string]bool{},
		routeAgency:  map[string]string{},
		shapeRoute:   map[string]string{},
		maxTripSeq:   map[string]int{},
		stopLocation: map[string]*latlon{},
	}

	// Checking the length of the 0th entry ensures we ignore the case where
	// BUS_ROUTE_FILTER was an empty string (resulting in []string{""}).
	// Possibly we want to check this with the conf package, but doing this for
	// now.
	if len(conf.Loader.RouteFilter) > 0 &&
		len(conf.Loader.RouteFilter[0]) > 0 {

		l.routeIDs = map[string]bool{}
		for _, v := range conf.Loader.RouteFilter {
			l.routeIDs[v] = true
		}
	}

	return &l
}

func (l *Loader) load() {
	l.loadRoutes()
	l.loadTrips()
	l.loadStopLocations()
	l.loadStopTimes()
	l.loadUniqueStop()
	l.loadCalendars()
	l.loadCalendarDates()
	l.loadShapes()

	l.updateRouteShapes()
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
	var i int
	f, fh := getcsv(l.dir, "routes.txt")
	defer fh.Close()

	header, err := f.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	routeIdx := find(header, "route_id")
	routeTypeIdx := find(header, "route_type")
	routeColorIdx := find(header, "route_color")
	routeTextColorIdx := find(header, "route_text_color")
	routeAgencyIdx := find(header, "agency_id")
	routeShortNameIdx := find(header, "route_short_name")
	routeLongNameIdx := find(header, "route_long_name")

	for i = 0; ; i++ {
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
		agencyID := rec[routeAgencyIdx]
		shortName := rec[routeShortNameIdx]
		longName := rec[routeLongNameIdx]

		r, err := models.NewRoute(
			route, routeType, routeColor, routeTextColor, agencyID,
			shortName, longName,
		)
		if err != nil {
			log.Fatalf("%v on line %v of routes.txt", err, i)
		}

		err = r.Save()
		if err != nil {
			log.Fatalf("%v on line %v of routes.txt", err, i)
		}

		l.routeAgency[route] = agencyID
	}

	log.Printf("loaded %v routes", i)

}

func (l *Loader) loadTrips() {
	var i int

	f, fh := getcsv(l.dir, "trips.txt")
	defer fh.Close()

	header, err := f.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	tripIdx := find(header, "trip_id")
	dirIdx := find(header, "direction_id")
	headIdx := find(header, "trip_headsign")
	serviceIdx := find(header, "service_id")
	routeIdx := find(header, "route_id")
	shapeIdx := find(header, "shape_id")

	for i = 0; ; i++ {
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

		id := rec[tripIdx]
		service := rec[serviceIdx]
		route := rec[routeIdx]
		shape := rec[shapeIdx]
		agency := l.routeAgency[route]

		if l.skipRoute(route) {
			continue
		}

		trip, err := models.NewTrip(
			id, route, agency, service, shape, rec[headIdx], direction,
		)
		if err != nil {
			log.Fatalf("%v on line %v of trips.txt", err, i)
		}

		l.trips[trip.TripID] = trip

		serviceObj := &models.Service{
			ID:      service,
			RouteID: route,
		}

		l.tripService[trip.TripID] = serviceObj

		if l.serviceRoute[service] == nil {
			l.serviceRoute[service] = map[string]bool{}
		}
		l.serviceRoute[service][route] = true

		err = trip.Save()
		if err != nil {
			log.Fatalf("%v on line %v of trips.txt", err, i)
		}

		l.tripRoute[id] = route
		l.shapeRoute[shape] = route

		if i%logp == 0 {
			log.Printf("loaded %v trips", i)
		}

	}

	log.Printf("loaded %v trips", i)

}

func (l *Loader) loadStopTimes() {
	var i int
	var err error

	// Read the unsorted file first so we can get headers index values
	stopTimesUnsorted, unsortedFh := getcsv(l.dir, "stop_times.txt")
	defer unsortedFh.Close()

	header, err := stopTimesUnsorted.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	stopIdx := find(header, "stop_id")
	tripIdx := find(header, "trip_id")
	arrivalIdx := find(header, "arrival_time")
	depatureIdx := find(header, "departure_time")
	sequenceIdx := find(header, "stop_sequence")

	// Create a file in the same dir that we've guaranteed is sorted
	// by trip_id and stop_sequence. It probably already is, but let's
	// be sure
	noFirstLine, err := ioutil.TempFile(l.dir, "")
	if err != nil {
		log.Fatal("can't create first line file", err)
	}
	defer noFirstLine.Close()
	defer os.Remove(noFirstLine.Name())

	sorted, err := ioutil.TempFile(l.dir, "")
	if err != nil {
		log.Fatal("can't create sorted file", err)
	}
	defer sorted.Close()
	defer os.Remove(sorted.Name())

	// Remove first line of file (we don't want the header to be sorted
	// in the middle of the file)
	cmd := exec.Command("tail", "-n", "+2", path.Join(l.dir, "stop_times.txt"))
	cmd.Stdout = noFirstLine
	err = cmd.Run()
	if err != nil {
		log.Fatal("can't create file with no first line", err)
	}
	err = noFirstLine.Close()
	if err != nil {
		log.Fatal("can't close no first line file", err)
	}

	// Sort primarily by trip then by sequence id
	// sort -s -t, -k1,1 -k5,5n
	cmd = exec.Command("sort",
		"-s",
		"-t,",
		fmt.Sprintf("-k%d,%d", tripIdx+1, tripIdx+1),
		fmt.Sprintf("-k%d,%dn", sequenceIdx+1, sequenceIdx+1),
		noFirstLine.Name(),
	)
	cmd.Stdout = sorted
	err = cmd.Run()
	if err != nil {
		log.Fatal("can't sort file", err, cmd)
	}
	err = sorted.Close()
	if err != nil {
		log.Fatal("can't close sorted file")
	}

	// Open the sorted file and process
	stopTimes, fh := getcsv(l.dir, path.Base(sorted.Name()))
	defer fh.Close()

	// read once to get max sequence ids
	for i := 0; ; i++ {
		rec, err := stopTimes.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("%v on line %v of stop_times.txt", err, i)
		}

		trip := rec[tripIdx]
		sequenceStr := rec[sequenceIdx]
		sequence, err := strconv.Atoi(sequenceStr)
		if err != nil {
			log.Fatalf("%v on line %v of stop_times.txt", err, i)
		}

		if sequence > l.maxTripSeq[trip] {
			l.maxTripSeq[trip] = sequence
		}
	}
	err = fh.Close()
	if err != nil {
		log.Fatal("can't close", err)
	}

	stopTimes, fh = getcsv(l.dir, path.Base(sorted.Name()))
	defer fh.Close()

	// read again for actual processing
	var rec []string
	var sst *models.ScheduledStopTime
	var stop string
	for i = 0; ; i++ {
		rec, err = stopTimes.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("%v on line %v of stop_times.txt", err, i)
		}

		stop = rec[stopIdx]
		trip := rec[tripIdx]
		arrivalStr := rec[arrivalIdx]
		departureStr := rec[depatureIdx]
		agencyID := l.routeAgency[l.tripRoute[trip]]
		sequenceStr := rec[sequenceIdx]
		sequence, err := strconv.Atoi(sequenceStr)
		if err != nil {
			log.Fatalf("%v on line %v of stop_times.txt", err, i)
		}

		l.stopTrips[stop] = append(l.stopTrips[stop], trip)

		service, exists := l.tripService[trip]
		if !exists {
			continue
		}

		lastStop := false
		maxSeq := l.maxTripSeq[trip]
		if maxSeq == sequence {
			lastStop = true
		}

		// Save the sst from the previous iteration
		if sst != nil {

			ll := l.stopLocation[stop]
			if ll == nil {
				log.Fatal("can't get lat lon", stop)
			}

			if sst.TripID == trip {
				sst.NextStopID.Scan(stop)
				sst.NextStopLat = ll.lat
				sst.NextStopLon = ll.lon
			} else {
				sst.NextStopID.Scan(nil)
				sst.NextStopLat = 0
				sst.NextStopLon = 0
			}

			err = sst.Save()
			if err != nil {
				log.Fatalf("%v on line %v of stop_times.txt", err, i)
			}
		}

		sst, err = models.NewScheduledStopTime(
			service.RouteID, stop, service.ID, arrivalStr, departureStr,
			agencyID, trip, sequence, lastStop,
		)
		if err != nil {
			log.Fatalf("%v on line %v of stop_times.txt", err, i)
		}

		if i%logp == 0 {
			log.Printf("loaded %v stop times", i)
		}
	}

	// Make sure we get the last stop
	if sst != nil {
		// This will always be nil
		sst.NextStopID.Scan(nil)

		err = sst.Save()
		if err != nil {
			log.Fatalf("%v on line %v of stop_times.txt", err, i)
		}
	}

	log.Printf("loaded %v stop times", i)
}

func (l *Loader) loadStopLocations() {
	var i int

	stops, fh := getcsv(l.dir, "stops.txt")
	defer fh.Close()

	header, err := stops.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	stopIdx := find(header, "stop_id")
	stopLatIdx := find(header, "stop_lat")
	stopLonIdx := find(header, "stop_lon")

	for i = 0; ; i++ {
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

		ll := &latlon{
			lat: stopLat,
			lon: stopLon,
		}

		l.stopLocation[rec[stopIdx]] = ll
	}

	log.Printf("loaded %v stops locations", i)
}

func (l *Loader) loadUniqueStop() {
	var i int

	stops, fh := getcsv(l.dir, "stops.txt")
	defer fh.Close()

	header, err := stops.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	stopIdx := find(header, "stop_id")
	stopNameIdx := find(header, "stop_name")
	stopLatIdx := find(header, "stop_lat")
	stopLonIdx := find(header, "stop_lon")

	for i = 0; ; i++ {
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
					StopID:  rec[stopIdx],
					Name:    rec[stopNameIdx],
					Lat:     stopLat,
					Lon:     stopLon,
					RouteID: l.tripRoute[trip],

					DirectionID: l.trips[trip].DirectionID,
					Headsign:    l.trips[trip].Headsign,
					AgencyID:    l.routeAgency[l.tripRoute[trip]],
				}

				err = obj.Save()
				if err != nil {
					log.Fatalf("%v on line %v of stops.txt", err, i)
				}
			}
		}

		if i%logp == 0 {
			log.Printf("loaded %v stops", i)
		}
	}

	log.Printf("loaded %v stops", i)
}

func (l *Loader) loadCalendarDates() {

	cal, fh := getcsv(l.dir, "calendar_dates.txt")
	defer fh.Close()

	header, err := cal.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	serviceIdx := find(header, "service_id")
	exceptionDateIdx := find(header, "date")
	exceptionTypeIdx := find(header, "exception_type")

	for i := 0; ; i++ {
		rec, err := cal.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("%v on line %v of calendar_dates.txt", err, i)
		}

		serviceId := rec[serviceIdx]

		exceptionDate, err := time.Parse(datefmt, rec[exceptionDateIdx])
		if err != nil {
			log.Fatalf("can't parse exception date %v %v",
				err, rec[exceptionDateIdx])
		}

		exceptionType, err := strconv.Atoi(rec[exceptionTypeIdx])
		if err != nil {
			log.Fatalf("can't parse exception type integer %v %v",
				err, rec[exceptionTypeIdx])
		}

		if !(exceptionType == models.ServiceAdded || exceptionType == models.ServiceRemoved) {
			log.Fatalf("invalid value for exception_type %v", exceptionType)
		}

		for route, _ := range l.serviceRoute[serviceId] {
			s := models.ServiceRouteException{
				AgencyID:      l.routeAgency[route],
				ServiceID:     serviceId,
				RouteID:       route,
				ExceptionDate: exceptionDate,
				ExceptionType: exceptionType,
			}

			err = s.Save()
			if err != nil {
				log.Fatalf("%v on line %v of calendar_dates.txt with %v", err, i, s)
			}
		}
	}
}

func (l *Loader) loadCalendars() {
	var i int

	cal, fh := getcsv(l.dir, "calendar.txt")
	defer fh.Close()

	header, err := cal.Read()
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

	for i = 0; ; i++ {
		rec, err := cal.Read()
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
					AgencyID:  l.routeAgency[route],
					Day:       day,
					StartDate: startDate,
					EndDate:   endDate,
				}

				err = srd.Save()
				if err != nil {
					log.Fatalf("%v on line %v of calendar.txt with %v", err, i, srd)
				}
			}
		}

		if i%logp == 0 {
			log.Printf("loaded %v calendars", i)
		}
	}

	log.Printf("loaded %v calendars", i)
}

func (l *Loader) loadShapes() {
	var i int

	shapes, fh := getcsv(l.dir, "shapes.txt")
	defer fh.Close()

	header, err := shapes.Read()
	if err != nil {
		log.Fatalf("unable to read header: %v", err)
	}

	idIDX := find(header, "shape_id")
	latIDX := find(header, "shape_pt_lat")
	lonIDX := find(header, "shape_pt_lon")
	seqIDX := find(header, "shape_pt_sequence")

	for i = 0; ; i++ {
		rec, err := shapes.Read()
		if err == io.EOF {
			break
		}

		lat, err := strconv.ParseFloat(
			strings.TrimSpace(rec[latIDX]), 64,
		)
		if err != nil {
			log.Fatalf("%v on line %v of shapes.txt", err, i)
		}

		lon, err := strconv.ParseFloat(
			strings.TrimSpace(rec[lonIDX]), 64,
		)
		if err != nil {
			log.Fatalf("%v on line %v of shapes.txt", err, i)
		}

		seq, err := strconv.ParseInt(
			strings.TrimSpace(rec[seqIDX]), 10, 32,
		)

		id := rec[idIDX]
		route := l.shapeRoute[id]
		if len(route) < 1 || l.skipRoute(route) {
			continue
		}

		agency := l.routeAgency[l.shapeRoute[id]]

		shape, err := models.NewShape(
			id, agency, int(seq), lat, lon,
		)
		err = shape.Save(etc.DBConn)
		if err != nil {
			log.Fatalf("%v on line %v of shapes.txt", err, i)
		}

		if i%logp == 0 {
			log.Printf("loaded %v shapes", i)
		}
	}

	log.Printf("loaded %v shapes", i)
}

// updateRouteShapes updates the route_shape table by identifying
// the "biggest" shapes typical for a route
func (l *Loader) updateRouteShapes() {
	var err error

	tx, err := etc.DBConn.Beginx()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			if err != nil {
				log.Println("can't commit route shapes", err)
			}
		} else {
			tx.Rollback()
			if err != nil {
				log.Println("can't rollback route shapes", err)
			}
		}
	}()

	// delete existing routes within a transaction (won't take effect
	// unless committed)
	err = models.DeleteRouteShapes(tx)
	if err != nil {
		log.Fatal(err)
	}

	// Get shapes ordered from smallest to largest
	routeShapes, err := models.GetRouteShapes(tx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("got %d route shapes", len(routeShapes))

	for _, rs := range routeShapes {
		// upsert each route so we end up with the most common
		err = rs.Save(tx)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Printf("saved %v", rs)
		}
	}

	// delete existing routes within a transaction (won't take effect
	// unless committed)
	err = models.DeleteFakeShapes(tx)
	if err != nil {
		log.Fatal(err)
	}

	// Get shapes ordered from smallest to largest
	fakeShapes, err := models.GetFakeRouteShapes(tx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("got %d fake shapes", len(fakeShapes))

	for _, fs := range fakeShapes {
		// upsert each route so we end up with the most common
		err = fs.Save(tx)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Printf("saved %v", fs)
		}
	}

}

// LoadOnce loads the files in conf.Loader.GTFSURLs, possibly filtering by the
// routes specified in conf.Loader.RouteFilter. If no filter is defined,
// it loads all data in the specified URLs.
func LoadOnce() {
	for _, url := range conf.Loader.GTFSURLs {
		if len(url) < 1 {
			continue
		}

		log.Printf("starting %v", url)

		// FIXME: do this in Go, need to make it integrated with loader
		dir, err := ioutil.TempDir(conf.Loader.TmpDir, "")
		if err != nil {
			log.Fatal(err)
		}
		cmd := exec.Command("wget", url, "-O", path.Join(dir, "file.zip"))
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

		cmd = exec.Command("unzip", path.Join(dir, "file.zip"), "-d", dir)
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

		err = prepare(url, dir)
		if err != nil {
			log.Fatal(err)
		}

		func() {
			log.Printf("loading: %v in %v", url, dir)
			defer os.RemoveAll(dir)

			t1 := time.Now()
			l := newLoader(dir)
			l.load()
			t2 := time.Now()

			log.Printf("took %v for %v", t2.Sub(t1), url)
		}()
	}

	// Update materialized views. Use a transaction for each one, because
	// we reset each view's primary ID sequence in a separate statement.
	for _, view := range views {
		func() {
			var err error

			tx, err := etc.DBConn.Beginx()
			if err != nil {
				log.Fatal("can't create tx to update view", view, err)
			}

			defer func() {
				if err == nil {
					commitErr := tx.Commit()
					if commitErr != nil {
						log.Fatal("error committing update to view", view, err)
					}
				} else {
					rollbackErr := tx.Rollback()
					if rollbackErr != nil {
						log.Fatal("error rolling back update to view", view, err)
					}
				}
			}()

			statements := []string{
				fmt.Sprintf("ALTER SEQUENCE %s_seq RESTART WITH 1", view),
				fmt.Sprintf("REFRESH MATERIALIZED VIEW CONCURRENTLY %s", view),
			}

			for _, statement := range statements {
				log.Println(statement)
				_, err = tx.Exec(statement)
				if err != nil {
					log.Println("can't exec", statement, err)
					return
				}
				log.Println("complete")
			}
		}()
	}
}

// LoadForever continuously runs LoadOnce, breaking for 24 hours between loads
func LoadForever() {
	for {
		LoadOnce()

		log.Printf("finished loading, sleeping for %v", loaderBreak)
		time.Sleep(loaderBreak)
	}
}

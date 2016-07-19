// Package bus_test runs full end-to-end tests of the bus system by
// running the loader against a small subset of live data and hitting
// the API, checking for sane results. Most settings will be read
// from the environment like the normal application, but
// $BUS_GTFS_URLS and $BUS_ROUTE_FILTER will be overidden by
// the tests.
package bus_test

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/brnstz/bus/api"
	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/loader"
	"github.com/kelseyhightower/envconfig"
)

var (
	// serverURL is the url of our test HTTP server
	serverURL string
)

// TestMain initializes/loads the database and starts an HTTP server to test
// against.
func TestMain(m *testing.M) {
	var err error
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	err = envconfig.Process("bus", &conf.DB)
	if err != nil {
		log.Fatal(err)
	}

	err = envconfig.Process("bus", &conf.Loader)
	if err != nil {
		log.Fatal(err)
	}

	err = envconfig.Process("bus", &conf.API)
	if err != nil {
		log.Fatal(err)
	}

	err = envconfig.Process("bus", &conf.Cache)
	if err != nil {
		log.Fatal(err)
	}

	err = envconfig.Process("bus", &conf.Partner)
	if err != nil {
		log.Fatal(err)
	}

	time.Local, err = time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}

	// Load the just subway and Brooklyn bus files
	conf.Loader.GTFSURLs = []string{
		"http://pub.brnstz.com.s3-website-us-east-1.amazonaws.com/bus/testdata/google_transit.zip",
		"http://pub.brnstz.com.s3-website-us-east-1.amazonaws.com/bus/testdata/google_transit_brooklyn.zip",
	}

	// Filter on a few routes for our tests
	conf.Loader.RouteFilter = []string{
		"G", "L", "B62", "B43", "B32",
	}

	conf.API.WebDir = "web/"

	// Get a db connection
	etc.DBConn = etc.MustDB()

	// Alow skipping the load if we trust our db is ok
	if os.Getenv("BUS_TEST_SKIP_LOADER") != "true" {
		// Load files once and return
		loader.LoadOnce()
	}

	// Create an HTTP server for our tests and set the URL
	server := httptest.NewServer(api.NewHandler())
	defer server.Close()
	serverURL = server.URL

	// Exit when it's over
	os.Exit(m.Run())
}

// hereResponse is the response to /api/here
type hereResponse struct {
	Stops []struct {
		Lat          float64
		Lon          float64
		Stop_ID      string
		Stop_Name    string
		Route_ID     string
		Headsign     string
		Direction_ID int
		Departures   []struct {
			Time time.Time
		}
		Dist float64
	}
}

// getJSON performs an HTTP get on the incoming URL and marshals
// the body of the response into v (which should be a pointer to
// something).
func getJSON(v interface{}, u string) error {
	resp, err := http.Get(u)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, v)
	if err != nil {
		return err
	}

	return nil
}

// TestScheduledSubway tests for scheduled subway times
func TestScheduledSubway(t *testing.T) {
	var resp hereResponse
	var err error

	params := url.Values{}
	now := time.Now().Add(-time.Minute * 5)

	expectedStop := "Greenpoint Av"
	expectedRoute := "G"

	// Manhattan Av. and Greenpoint Av. in Brooklyn
	params.Set("lat", "40.731324619044514")
	params.Set("lon", "-73.95446261823963")
	params.Set("sw_lat", "40.73059087592777")
	params.Set("sw_lon", "-73.95548990429234")
	params.Set("ne_lat", "40.73205835407014")
	params.Set("ne_lon", "-73.95343533218693")

	err = getJSON(&resp, serverURL+"/api/here?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for G train test", err)
	}

	if len(resp.Stops) != 2 {
		t.Fatalf("expected %v results but got %v", 2, len(resp.Stops))
	}

	// Check each result
	for _, v := range resp.Stops {
		if v.Stop_Name != expectedStop {
			t.Errorf("expected %v stop_name but got %v", expectedStop, v.Stop_Name)
		}

		if v.Route_ID != expectedRoute {
			t.Errorf("expected %v route_id but got %v", expectedRoute, v.Route_ID)
		}

		if len(v.Departures) < 1 {
			t.Errorf("expected at least one scheduled departure but got none in %#v", v)
		}

		// Check that scheduled times are in the future
		for _, d := range v.Departures {
			if d.Time.Before(now) {
				t.Errorf("expected scheduled time %v would be after or equal to %v but it was not", v.Departures, now)
			}
		}
	}
}

// TestLiveSubway checks for live subway times
func TestLiveSubway(t *testing.T) {
	var resp hereResponse
	var err error

	params := url.Values{}
	now := time.Now().Add(-time.Minute * 5)

	expectedStop := "Bedford Av"
	expectedRoute := "L"

	// Bedford Av. and N. 7th St. in Brooklyn
	params.Set("lat", "40.71736129085624")
	params.Set("lon", "-73.95682848743489")
	params.Set("sw_lat", "40.71662739378481")
	params.Set("sw_lon", "-73.9578557734876")
	params.Set("ne_lat", "40.71809517983715")
	params.Set("ne_lon", "-73.95580120138219")

	err = getJSON(&resp, serverURL+"/api/here?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for L train test", err)
	}

	// Check each result
	count := 0
	for _, v := range resp.Stops {
		// ignore results not for our expected route
		if v.Route_ID != expectedRoute {
			continue
		}

		// Count how many stops we get for our expected route
		count++

		if v.Stop_Name != expectedStop {
			t.Errorf("expected %v stop_name but got %v", expectedStop, v.Stop_Name)
		}

		if len(v.Departures) < 1 {
			t.Errorf("expected at least one live departure but got none in %#v", v)
		}

		// Check that live times are in the future
		for _, d := range v.Departures {
			if d.Time.Before(now) {
				t.Errorf("expected live time %v would be after or equal to %v but it was not", d.Time, now)
			}
		}
	}

	if count != 2 {
		t.Fatalf("expected %v results but got %v", 2, len(resp.Stops))
	}

}

// TestLiveBus checks for live bus results
func TestLiveBus(t *testing.T) {
	var resp hereResponse
	var err error

	params := url.Values{}

	// Jackson Av. and 11th St. in Queens
	params.Set("lat", "40.74235145797162")
	params.Set("lon", "-73.95134031772612")
	params.Set("sw_lat", "40.74088420686416")
	params.Set("sw_lon", "-73.95339488983153")
	params.Set("ne_lat", "40.7438186767128")
	params.Set("ne_lon", "-73.94928574562073")

	err = getJSON(&resp, serverURL+"/api/here?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for B62, B32 bus test", err)
	}

	// We should get results for at least the B62
	if len(resp.Stops) < 2 {
		t.Fatalf("expected at least %v results but got %v", 2, len(resp.Stops))
	}

	// Check each bus result
	for _, v := range resp.Stops {
		if v.Route_ID != "B62" && v.Route_ID != "B32" {
			continue
		}

		// FIXME: some bus routes won't always have live departures, may need to
		// pick different bus or make this a warning
		/*
			if len(v.Live) < 1 {
				t.Errorf("expected at least one live departure but got none in %#v", v)
			}
		*/

		// Check that
		for _, d := range v.Departures {
			if d.Time.IsZero() {
				t.Errorf("empty time identified in live departure in %#v", v)
			}
		}
	}

}

// TestScheduledBus checks for scheduled bus results
func TestScheduledBus(t *testing.T) {
	var resp hereResponse
	var err error

	params := url.Values{}
	now := time.Now().Add(-time.Minute * 5)

	expectedStop := "BOX ST/MANHATTAN AV"
	expectedRoute := "B43"

	// Box St. and Manhattan Av. in Brooklyn
	params.Set("lat", "40.73712490501606")
	params.Set("lon", "-73.95574549209414")
	params.Set("sw_lat", "40.735657538620636")
	params.Set("sw_lon", "-73.95780006419955")
	params.Set("ne_lat", "40.73859223904608")
	params.Set("ne_lon", "-73.95369091998873")

	err = getJSON(&resp, serverURL+"/api/here?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for B43 bus test", err)
	}

	// Check each result
	for _, v := range resp.Stops {
		if v.Stop_Name != expectedStop {
			t.Errorf("expected %v stop_name but got %v", expectedStop, v.Stop_Name)
		}

		if v.Route_ID != expectedRoute {
			t.Errorf("expected %v route_id but got %v", expectedRoute, v.Route_ID)
		}

		// Check that scheduled times are in the future
		for _, d := range v.Departures {
			if d.Time.Before(now) {
				t.Errorf("expected scheduled time %v would be after or equal to %v but it was not", d.Time, now)
			}
		}
	}
}

type tripResp struct {
	ID           string
	Shape_Points []struct {
		Lat float64
		Lon float64
	}
}

func TestTrip(t *testing.T) {
	trip := tripResp{}

	params := url.Values{}
	params.Set("agency_id", "MTA NYCT")
	params.Set("trip_id", "B20160612SAT_083700_G..N13R")
	params.Set("route_id", "G")

	u := serverURL + "/api/trip?" + params.Encode()

	err := getJSON(&trip, u)
	if err != nil {
		t.Fatal("can't get API response for trip", err)
	}

	if len(trip.Shape_Points) != 520 {
		t.Fatal("expected 520 shape points but got:", len(trip.Shape_Points))
	}

	for _, shape := range trip.Shape_Points {
		if shape.Lat < 40 || shape.Lat > 41 {
			t.Fatal("expected latitiude around 40 but got:", shape.Lat)
		}

		if shape.Lon > -73 || shape.Lon < -74 {
			t.Fatal("expected longitude around -73 but got:", shape.Lon)
		}
	}
}

func TestStopSpeed(t *testing.T) {

	resp := hereResponse{}
	params := url.Values{}

	// Large area in midtown east
	params.Set("lat", "40.75245875985305")
	params.Set("lon", "-73.97781372070312")
	params.Set("sw_lat", "40.74657419673222")
	params.Set("sw_lon", "-73.99798393249512")
	params.Set("ne_lat", "40.758342802212724")
	params.Set("ne_lon", "-73.95764350891112")
	params.Add("route_type", "0")
	params.Add("route_type", "1")
	params.Add("route_type", "2")

	t1 := time.Now()
	err := getJSON(&resp, serverURL+"/api/here?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for speed test", err, resp)
	}
	totalTime := time.Now().Sub(t1)
	if totalTime > 1*time.Second {
		t.Fatal("too slow")
	}
}

// TestHereYesterday tests we can get departures that start yesterday and today
func TestHereYesterday(t *testing.T) {
	var resp hereResponse
	var err error

	params := url.Values{}

	// Manhattan Av. and Greenpoint Av. in Brooklyn
	params.Set("lat", "40.731324619044514")
	params.Set("lon", "-73.95446261823963")
	params.Set("sw_lat", "40.73059087592777")
	params.Set("sw_lon", "-73.95548990429234")
	params.Set("ne_lat", "40.73205835407014")
	params.Set("ne_lon", "-73.95343533218693")
	params.Set("now", "2016-07-18 00:00:01")

	err = getJSON(&resp, serverURL+"/api/here?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for G train test", err)
	}

	expectedTimes := map[string][]string{
		"G26N": []string{
			"2016-07-18T00:01:00-04:00",
			"2016-07-18T00:16:00-04:00",
			"2016-07-18T00:35:00-04:00",
			"2016-07-18T00:55:00-04:00",
			"2016-07-18T01:15:00-04:00",
		},

		"G26S": []string{
			"2016-07-18T00:02:30-04:00",
			"2016-07-18T00:17:30-04:00",
			"2016-07-18T00:37:30-04:00",
			"2016-07-18T00:57:30-04:00",
			"2016-07-18T01:17:30-04:00",
		},
	}

	for _, stop := range resp.Stops {
		for i, departure := range stop.Departures {
			expectedTime, err := time.Parse(time.RFC3339, expectedTimes[stop.Stop_ID][i])
			if err != nil {
				t.Fatal(expectedTimes[stop.Stop_ID][i], err)
			}

			if !departure.Time.Equal(expectedTime) {
				t.Fatalf("actual departure time %v did not match expected %v", departure.Time, expectedTime)
			}
		}
	}
}

// TestHereToday tests we can get departures that are in the middle of the day,
// with nothing from yesterday or today
func TestHereToday(t *testing.T) {
	var resp hereResponse
	var err error

	params := url.Values{}

	// Manhattan Av. and Greenpoint Av. in Brooklyn
	params.Set("lat", "40.731324619044514")
	params.Set("lon", "-73.95446261823963")
	params.Set("sw_lat", "40.73059087592777")
	params.Set("sw_lon", "-73.95548990429234")
	params.Set("ne_lat", "40.73205835407014")
	params.Set("ne_lon", "-73.95343533218693")
	params.Set("now", "2016-07-18 12:05:00")

	err = getJSON(&resp, serverURL+"/api/here?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for G train test", err)
	}

	expectedTimes := map[string][]string{
		"G26N": []string{
			"2016-07-18T12:06:30-04:00",
			"2016-07-18T12:16:30-04:00",
			"2016-07-18T12:26:30-04:00",
			"2016-07-18T12:36:30-04:00",
			"2016-07-18T12:46:30-04:00",
		},

		"G26S": []string{
			"2016-07-18T12:08:00-04:00",
			"2016-07-18T12:19:00-04:00",
			"2016-07-18T12:30:00-04:00",
			"2016-07-18T12:40:00-04:00",
			"2016-07-18T12:50:00-04:00",
		},
	}

	for _, stop := range resp.Stops {
		for i, departure := range stop.Departures {
			expectedTime, err := time.Parse(time.RFC3339, expectedTimes[stop.Stop_ID][i])
			if err != nil {
				t.Fatal(expectedTimes[stop.Stop_ID][i], err)
			}

			if !departure.Time.Equal(expectedTime) {
				t.Fatalf("actual departure time %v did not match expected %v", departure.Time, expectedTime)
			}
		}
	}
}

// TestHereTomorrow tests we can get departures that start yesterday and today
func TestHereTomorrow(t *testing.T) {
	var resp hereResponse
	var err error

	params := url.Values{}

	// Manhattan Av. and Greenpoint Av. in Brooklyn
	params.Set("lat", "40.731324619044514")
	params.Set("lon", "-73.95446261823963")
	params.Set("sw_lat", "40.73059087592777")
	params.Set("sw_lon", "-73.95548990429234")
	params.Set("ne_lat", "40.73205835407014")
	params.Set("ne_lon", "-73.95343533218693")
	params.Set("now", "2016-07-18 23:47:01")

	err = getJSON(&resp, serverURL+"/api/here?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for G train test", err)
	}

	expectedTimes := map[string][]string{
		"G26N": []string{
			"2016-07-19T00:01:00-04:00",
			"2016-07-19T00:18:30-04:00",
			"2016-07-19T00:37:30-04:00",
			"2016-07-19T00:57:30-04:00",
			"2016-07-19T01:17:30-04:00",
		},

		"G26S": []string{
			"2016-07-19T00:01:00-04:00",
			"2016-07-19T00:17:30-04:00",
			"2016-07-19T00:37:30-04:00",
			"2016-07-19T00:57:30-04:00",
			"2016-07-19T01:17:30-04:00",
		},
	}

	for _, stop := range resp.Stops {
		for i, departure := range stop.Departures {
			expectedTime, err := time.Parse(time.RFC3339, expectedTimes[stop.Stop_ID][i])
			if err != nil {
				t.Fatal(expectedTimes[stop.Stop_ID][i], err)
			}

			if !departure.Time.Equal(expectedTime) {
				t.Fatalf("actual departure time %v did not match expected %v", departure.Time, expectedTime)
			}
		}
	}
}

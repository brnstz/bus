// Package bus_test runs full end-to-end tests of the bus system by
// running the loader against a small subset of live data and hitting
// the API, checking for sane results. Most settings will be read
// from the environment like the normal application, but
// $BUS_GTFS_URLS and $BUS_ROUTE_FILTER will be overidden by
// the tests.
package bus_test

import (
	"encoding/json"
	"fmt"
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

	time.Local, err = time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}

	// Load the just subway and Brooklyn bus files
	conf.Loader.GTFSURLs = []string{
		"http://web.mta.info/developers/data/nyct/subway/google_transit.zip",
		"http://web.mta.info/developers/data/nyct/bus/google_transit_brooklyn.zip",
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

// stopResponse is the response to /api/stops
type stopResponse struct {
	Stops []struct {
		Lat          float64
		Lon          float64
		Stop_ID      string
		Stop_Name    string
		Route_ID     string
		Headsign     string
		Direction_ID int
		Live         []struct {
			Time time.Time
		}

		Scheduled []struct {
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
	var resp stopResponse
	var err error

	params := url.Values{}
	now := time.Now().Add(-time.Minute * 5)

	expectedStop := "Greenpoint Av"
	expectedRoute := "G"

	// Manhattan Av. and Greenpoint Av. in Brooklyn
	params.Set("lat", "40.730202")
	params.Set("lon", "-73.9564682")
	params.Set("miles", "0.1")
	params.Set("filter", "subway")

	err = getJSON(&resp, serverURL+"/api/stops?"+params.Encode())
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

		if len(v.Scheduled) < 1 {
			t.Errorf("expected at least one scheduled departure but got none in %#v", v)
		}

		// Check that scheduled times are in the future
		for _, d := range v.Live {
			if d.Time.Before(now) {
				t.Errorf("expected scheduled time %v would be after or equal to %v but it was not", v.Scheduled, now)
			}
		}
	}
}

// TestLiveSubway checks for live subway times
func TestLiveSubway(t *testing.T) {
	var resp stopResponse
	var err error

	params := url.Values{}
	now := time.Now().Add(-time.Minute * 5)

	expectedStop := "Bedford Av"
	expectedRoute := "L"

	// Bedford Av. and N. 7th St. in Brooklyn
	params.Set("lat", "40.717304")
	params.Set("lon", "-73.956872")
	params.Set("miles", "0.01")

	err = getJSON(&resp, serverURL+"/api/stops?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for L train test", err)
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

		if len(v.Live) < 1 {
			t.Errorf("expected at least one live departure but got none in %#v", v)
		}

		// Check that live times are in the future
		for _, d := range v.Live {
			if d.Time.Before(now) {
				t.Errorf("expected live time %v would be after or equal to %v but it was not", d.Time, now)
			}
		}
	}
}

// TestLiveBus checks for live bus results
func TestLiveBus(t *testing.T) {
	var resp stopResponse
	var err error

	params := url.Values{}

	// Jackson Av. and 11th St. in Queens
	params.Set("lat", "40.7422511")
	params.Set("lon", "-73.9515471")
	params.Set("miles", "0.1")

	err = getJSON(&resp, serverURL+"/api/stops?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for B62, B32 bus test", err)
	}

	// We should get results for both B62 and B32
	if len(resp.Stops) != 4 {
		t.Fatalf("expected %v results but got %v", 4, len(resp.Stops))
	}

	// Check each result
	for _, v := range resp.Stops {

		// FIXME: some bus routes won't always have live departures, may need to
		// pick different bus or make this a warning
		/*
			if len(v.Live) < 1 {
				t.Errorf("expected at least one live departure but got none in %#v", v)
			}
		*/

		// Check that
		for _, d := range v.Live {
			if d.Time.IsZero() {
				t.Errorf("empty time identified in live departure in %#v", v)
			}
		}
	}

}

// TestScheduledBus checks for scheduled bus results
func TestScheduledBus(t *testing.T) {
	var resp stopResponse
	var err error

	params := url.Values{}
	now := time.Now().Add(-time.Minute * 5)

	expectedStop := "BOX ST/MANHATTAN AV"
	expectedRoute := "B43"

	// Box St. and Manhattan Av. in Brooklyn
	params.Set("lat", "40.7373215")
	params.Set("lon", "-73.9563212")
	params.Set("miles", "0.1")

	err = getJSON(&resp, serverURL+"/api/stops?"+params.Encode())
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
		for _, d := range v.Scheduled {
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

	agencyID := "MTA NYCT"
	tripID := "B20151206SAT_083700_G..N13R"
	routeID := "G"

	u := fmt.Sprintf("%s/api/agencies/%s/routes/%s/trips/%s",
		serverURL, url.QueryEscape(agencyID), url.QueryEscape(routeID),
		url.QueryEscape(tripID),
	)
	log.Println(u)

	err := getJSON(&trip, u)
	if err != nil {
		t.Fatal("can't get API response for trip", err)
	}

	log.Println(len(trip.Shape_Points))

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

type routeResp struct {
	Updated_At time.Time
	Routes     []struct {
		Route_ID     string
		Route_Color  string
		Route_Shapes []struct {
			Shapes []struct {
				Lat float64
				Lon float64
				Seq int
			}
		}
	}
}

func TestRoutes(t *testing.T) {
	rr := routeResp{}

	params := url.Values{}

	params.Set("lat", "40.7373215")
	params.Set("lon", "-73.9563212")
	params.Set("miles", "0.5")

	u := fmt.Sprintf("%s/api/routes?%s", serverURL, params.Encode())

	err := getJSON(&rr, u)
	if err != nil {
		t.Fatal("can't get API response for routes", err)
	}

	if len(rr.Routes) < 5 {
		t.Fatalf("expected at least 5 routes but got %v", len(rr.Routes))
	}

	for _, route := range rr.Routes {
		var expColor string
		var skip bool

		switch route.Route_ID {
		case "G":
			expColor = "#6CBE45"
		case "L":
			expColor = "#A7A9AC"
		case "B62":
			expColor = "#00AEEF"
		case "B43":
			expColor = "#EE352E"
		case "B32":
			expColor = "#006CB7"
		default:
			skip = true
		}

		if skip {
			continue
		}

		// Expect at least 1 shape
		if len(route.Route_Shapes) < 1 {
			t.Fatalf("at least 1 shapes but got %v", len(route.Route_Shapes))
		}

		// Expect at least 10 points in each route_shape
		for _, rs := range route.Route_Shapes {
			if len(rs.Shapes) < 10 {
				t.Fatalf("expected at least 10 points but got %v", len(rs.Shapes))
			}
		}

		if expColor != route.Route_Color {
			t.Fatalf("expected %v color but got %v", expColor, route.Route_Color)
		}
	}
}

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

	// Load the just subway and Brooklyn bus files
	conf.Loader.GTFSURLs = []string{
		"http://web.mta.info/developers/data/nyct/subway/google_transit.zip",
		"http://web.mta.info/developers/data/nyct/bus/google_transit_brooklyn.zip",
	}

	// Filter on a few routes for our tests
	conf.Loader.RouteFilter = []string{
		"G", "L", "B62", "B43", "B32",
	}

	// Get a db connection
	etc.DBConn = etc.MustDB()

	// Load files once and return
	loader.LoadOnce()

	// Create an HTTP server for our tests and set the URL
	server := httptest.NewServer(api.NewHandler())
	defer server.Close()
	serverURL = server.URL

	// Exit when it's over
	os.Exit(m.Run())
}

// stopResults is the response to /api/v2/stops
type stopResults struct {
	Results []struct {
		Route struct {
			Route_ID     string
			Headsign     string
			Direction_ID string
		}

		Stop struct {
			Dist      string
			Lat       float64
			Lon       float64
			Stop_ID   string
			Stop_Name string
		}

		Departures struct {
			Live []struct {
				Time time.Time
			}

			Scheduled []struct {
				Time time.Time
			}
		}
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
	var res stopResponse
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

	err = getJSON(&res, serverURL+"/api/v2/stops?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for G train test", err)
	}

	if len(res) != 2 {
		t.Fatalf("expected %v results but got %v", 2, len(res))
	}

	// Check each result
	for _, v := range res {
		if v.StopName != expectedStop {
			t.Errorf("expected %v stop_name but got %v", expectedStop, v.StopName)
		}

		if v.RouteID != expectedRoute {
			t.Errorf("expected %v route_id but got %v", expectedRoute, v.RouteID)
		}

		if len(v.Scheduled) < 1 {
			t.Errorf("expected at least one scheduled departure but got none in %#v", v)
		}

		// Check that scheduled times are in the future
		for _, d := range v.Scheduled {
			if d.Time.Before(now) {
				t.Errorf("expected scheduled time %v would be after or equal to %v but it was not", v.Scheduled, now)
			}
		}
	}
}

// TestLiveSubway checks for live subway times
func TestLiveSubway(t *testing.T) {
	var res stopResponse
	var err error

	params := url.Values{}
	now := time.Now().Add(-time.Minute * 5)

	expectedStop := "Bedford Av"
	expectedRoute := "L"

	// Bedford Av. and N. 7th St. in Brooklyn
	params.Set("lat", "40.717304")
	params.Set("lon", "-73.956872")
	params.Set("miles", "0.01")

	err = getJSON(&res, serverURL+"/api/v2/stops?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for L train test", err)
	}

	if len(res) != 2 {
		t.Fatalf("expected %v results but got %v", 2, len(res))
	}

	// Check each result
	for _, v := range res {
		if v.StopName != expectedStop {
			t.Errorf("expected %v stop_name but got %v", expectedStop, v.StopName)
		}

		if v.RouteID != expectedRoute {
			t.Errorf("expected %v route_id but got %v", expectedRoute, v.RouteID)
		}

		if len(v.Live) < 1 {
			t.Errorf("expected at least one live departure but got none in %#v", v)
		}

		// Check that live times are in the future
		for _, d := range v.Live {
			if d.Time.Before(now) {
				t.Errorf("expected live time %v would be after or equal to %v but it was not", v.Live, now)
			}
		}
	}
}

// TestLiveBus checks for live bus results
func TestLiveBus(t *testing.T) {
	var res stopResponse
	var err error

	params := url.Values{}

	// Jackson Av. and 11th St. in Queens
	params.Set("lat", "40.7422511")
	params.Set("lon", "-73.9515471")
	params.Set("miles", "0.1")

	err = getJSON(&res, serverURL+"/api/v2/stops?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for B62, B32 bus test", err)
	}

	// We should get results for both B62 and B32
	if len(res) != 4 {
		t.Fatalf("expected %v results but got %v", 4, len(res))
	}

	// Check each result
	for _, v := range res {

		// FIXME: some bus routes won't always have live departures, may need to
		// pick different bus or make this a warning
		/*
			if len(v.Live) < 1 {
				t.Errorf("expected at least one live departure but got none in %#v", v)
			}
		*/

		// Check that description field is filled in
		for _, d := range v.Live {
			if len(d.Desc) < 1 {
				t.Errorf("empty description identified in live departure in %#v", v)
			}
		}
	}

}

func TestScheduledBus(t *testing.T) {
	var res stopResponse
	var err error

	params := url.Values{}
	now := time.Now().Add(-time.Minute * 5)

	expectedStop := "BOX ST/MANHATTAN AV"
	expectedRoute := "B43"

	// Box St. and Manhattan Av. in Brooklyn
	params.Set("lat", "40.7373215")
	params.Set("lon", "-73.9563212")
	params.Set("miles", "0.1")

	err = getJSON(&res, serverURL+"/api/v2/stops?"+params.Encode())
	if err != nil {
		t.Fatal("can't get API response for B43 bus test", err)
	}

	// Check each result
	for _, v := range res {
		if v.StopName != expectedStop {
			t.Errorf("expected %v stop_name but got %v", expectedStop, v.StopName)
		}

		if v.RouteID != expectedRoute {
			t.Errorf("expected %v route_id but got %v", expectedRoute, v.RouteID)
		}

		// Check that scheduled times are in the future
		for _, d := range v.Scheduled {
			if d.Time.Before(now) {
				t.Errorf("expected scheduled time %v would be after or equal to %v but it was not", v.Scheduled, now)
			}
		}
	}
}

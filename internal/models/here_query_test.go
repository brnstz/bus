package models

import (
	"strings"
	"testing"
)

func makehq(serviceIDs []string) HereQuery {
	return HereQuery{
		MidLat: 40.75245875985305,
		MidLon: -73.97781372070312,

		SWLat: 40.74657419673222,
		SWLon: -73.99798393249512,
		NELat: 40.758342802212724,
		NELon: -73.95764350891112,

		ServiceIDs: serviceIDs,

		// Between midnight and 3am let's say
		DepartureMin: 0,
		DepartureMax: 10800,
	}

}

func TestHereQuery(t *testing.T) {
	todayIDs := []string{"CA_C6-Saturday", "B20160612SAT"}

	hq := makehq(todayIDs)

	err := hq.Initialize()
	if err != nil {
		t.Fatal(err)
	}

	// Just that we did our service ids correctly
	if !strings.Contains(hq.Query, "service_id IN ('CA_C6-Saturday','B20160612SAT')") {
		t.Fatal("can't find today service ids in clause in string", hq.Query)
	}

}

// TestHereQueryBlank ensures that a blank list of service ids is encoded correctly
func TestHereQueryBlank(t *testing.T) {

	ydayIDs := []string{}

	hq := makehq(ydayIDs)

	err := hq.Initialize()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(hq.Query, "service_id IN ('')") {
		t.Fatal("can't find blank service ids in clause in string", hq.Query)
	}
}

// TestHereQueryEscape ensures that service ids with single quotes are escaped correctly
func TestHereQueryEscape(t *testing.T) {

	todayIDs := []string{"CA'_C6-Satur'''day", "B20''''160612SAT", ""}

	hq := makehq(todayIDs)

	err := hq.Initialize()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(hq.Query, "service_id IN ('CA''_C6-Satur''''''day','B20''''''''160612SAT','')") {
		t.Fatal("can't find escaped service ids in clause in string", hq.Query)
	}

}

package models

import (
	"strings"
	"testing"
)

func makehq(todayIDs, ydayIDs []string) HereQuery {
	return HereQuery{
		MidLat: 40.75245875985305,
		MidLon: -73.97781372070312,

		SWLat: 40.74657419673222,
		SWLon: -73.99798393249512,
		NELat: 40.758342802212724,
		NELon: -73.95764350891112,

		TodayServiceIDs:     todayIDs,
		YesterdayServiceIDs: ydayIDs,

		// Between midnight and 3am let's say
		TodayDepartureMin: 0,
		TodayDepartureMax: 10800,

		YesterdayDepartureMin: 82800,
		YesterdayDepartureMax: 99999,
	}

}

func TestHereQuery(t *testing.T) {
	todayIDs := []string{"CA_C6-Saturday", "B20160612SAT"}
	ydayIDs := []string{"A20160612WKD", "CH_C6-Weekday-SDon"}

	hq := makehq(todayIDs, ydayIDs)

	err := hq.Initialize()
	if err != nil {
		t.Fatal(err)
	}

	// Just that we did our service ids correctly
	if !strings.Contains(hq.Query, "service_id IN ('CA_C6-Saturday','B20160612SAT')") {
		t.Fatal("can't find today service ids in clause in string", hq.Query)
	}

	if !strings.Contains(hq.Query, "service_id IN ('A20160612WKD','CH_C6-Weekday-SDon')") {
		t.Fatal("service_ids don't appear to be correct")
	}

}

// TestHereQueryBlank ensures that a blank list of service ids is encoded correctly
func TestHereQueryBlank(t *testing.T) {

	todayIDs := []string{"CA_C6-Saturday", "B20160612SAT"}
	ydayIDs := []string{}

	hq := makehq(todayIDs, ydayIDs)

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
	ydayIDs := []string{}

	hq := makehq(todayIDs, ydayIDs)

	err := hq.Initialize()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(hq.Query, "service_id IN ('CA''_C6-Satur''''''day','B20''''''''160612SAT','')") {
		t.Fatal("can't find escaped service ids in clause in string", hq.Query)
	}

}

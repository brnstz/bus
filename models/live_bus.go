package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

var (
	busKey = os.Getenv("MTA_BUS_TIME_API_KEY")
	vmURL  = "http://bustime.mta.info/api/siri/vehicle-monitoring.json"
)

type Call struct {
	Extensions struct {
		Distances struct {
			CallDistanceAlongRoute float64
			DistanceFromCall       float64
			PresentableDistance    string
			StopsFromCall          int
		}
	}
	StopPointRef string
}

type CallSlice []Call

func (c CallSlice) Len() int {
	return len(c)
}

func (c CallSlice) Less(i, j int) bool {
	return c[i].Extensions.Distances.DistanceFromCall <
		c[j].Extensions.Distances.DistanceFromCall
}

func (c CallSlice) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

type journey struct {
	DirectionRef  string
	LineRef       string
	MonitoredCall Call
	OnwardCalls   struct {
		OnwardCall []Call
	}
}

type siriResp struct {
	Siri struct {
		ServiceDelivery struct {
			VehicleMonitoringDelivery []struct {
				ResponseTimestamp time.Time
				ValidUntil        time.Time
				VehicleActivity   []struct {
					MonitoredVehicleJourney journey
					RecordedAtTime          time.Time
				}
			}
		}
	}
}

func GetCallsByRouteStop(route, dir, stop string) (calls CallSlice, err error) {
	lineRef := fmt.Sprint("MTA NYCT_", route)
	stopPointRef := fmt.Sprint("MTA_", stop)

	q := url.Values{}
	q.Set("key", busKey)
	q.Set("DirectionRef", dir)
	q.Set("VehicleMonitoringDetailLevel", "calls")
	q.Set("LineRef", lineRef)
	u := fmt.Sprint(vmURL, "?", q.Encode())
	log.Println(u)

	resp, err := http.Get(u)
	if err != nil {
		log.Println("can't get vehicles for route", err, u)
		return
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("can't read body", err)
		return
	}

	sr := siriResp{}
	err = json.Unmarshal(b, &sr)
	if err != nil {
		log.Println("can't get unmarshal siriresp", err)
		return
	}

	vmd := sr.Siri.ServiceDelivery.VehicleMonitoringDelivery

	log.Println("hello 1")
	if len(vmd) > 0 {
		log.Println("hello 2")
		log.Printf("%+v", vmd)
		for _, act := range vmd[0].VehicleActivity {

			log.Println("hello 3")
			curCall := act.MonitoredVehicleJourney.MonitoredCall
			log.Println("current", curCall.StopPointRef, stopPointRef)
			if curCall.StopPointRef == stopPointRef {
				calls = append(calls, curCall)
			}

			for _, oc := range act.MonitoredVehicleJourney.OnwardCalls.OnwardCall {
				log.Println("onward", oc.StopPointRef, stopPointRef)
				if oc.StopPointRef == stopPointRef {
					calls = append(calls, oc)
				}
			}

		}
	}

	return
}

package realtime

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
)

var (
	vmURL = "http://bustime.mta.info/api/siri/vehicle-monitoring.json"
)

type mtaNYCBus struct{}

func (_ mtaNYCBus) Departures(route models.Route, stop models.Stop) (d Departures, err error) {
	lineRef := fmt.Sprint("MTA NYCT_", stop.RouteID)
	stopPointRef := fmt.Sprint("MTA_", stop.ID)

	q := url.Values{}
	q.Set("key", conf.API.BustimeAPIKey)
	q.Set("DirectionRef", stop.DirectionID)
	q.Set("VehicleMonitoringDetailLevel", "calls")
	q.Set("LineRef", lineRef)
	u := fmt.Sprint(vmURL, "?", q.Encode())

	b, err := etc.RedisCache(u)
	if err != nil {
		log.Println("can't get live buses", err)
		return
	}

	sr := siriResp{}
	err = json.Unmarshal(b, &sr)
	if err != nil {
		log.Println("can't get unmarshal siriresp", err)
		return
	}

	vmd := sr.Siri.ServiceDelivery.VehicleMonitoringDelivery

	if len(vmd) > 0 {
		for _, act := range vmd[0].VehicleActivity {

			for _, oc := range act.MonitoredVehicleJourney.OnwardCalls.OnwardCall {
				if oc.StopPointRef == stopPointRef {
					tripID := act.MonitoredVehicleJourney.FramedVehicleJourneyRef.DatedVehicleJourneyRef
					// remove "MTA NYCT_" from front of string
					if len(tripID) > 9 {
						tripID = tripID[9:]
					}
					d = append(d, &Departure{
						Time:   oc.ExpectedDepartureTime,
						TripID: tripID,
					})
				}
			}

		}
	}

	return
}

type call struct {
	ExpectedDepartureTime time.Time
	Extensions            struct {
		Distances struct {
			CallDistanceAlongRoute float64
			DistanceFromCall       float64
			PresentableDistance    string
			StopsFromCall          int
		}
	}
	StopPointRef string
}

type journey struct {
	DirectionRef string
	LineRef      string

	FramedVehicleJourneyRef struct {
		DatedVehicleJourneyRef string
	}

	OnwardCalls struct {
		OnwardCall []Call
	}

	// MonitoredCall is the current stop of the bus, but this
	// info appears to be duped in OnwardCall.
	// MonitoredCall Call
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

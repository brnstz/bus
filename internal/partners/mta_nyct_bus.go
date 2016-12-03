package partners

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
)

var (
	vmURL = "http://bustime.mta.info/api/siri/vehicle-monitoring.json"
)

type mtaNYCBus struct{}

func (p mtaNYCBus) IsLive() bool {
	return true
}

func (p mtaNYCBus) getURL(agencyID, routeID string, directionID int) string {
	lineRef := fmt.Sprintf("%v_%v", agencyID, routeID)

	q := url.Values{}
	q.Set("key", conf.Partner.BustimeAPIKey)
	q.Set("DirectionRef", strconv.Itoa(directionID))
	q.Set("VehicleMonitoringDetailLevel", "calls")
	q.Set("LineRef", lineRef)

	return vmURL + "?" + q.Encode()
}

func (p mtaNYCBus) Precache(agencyID, routeID string, directionID int) error {
	k := fmt.Sprintf("%v|%v|%v", agencyID, routeID, directionID)

	u := p.getURL(agencyID, routeID, directionID)

	_, err := etc.RedisCacheURL(u)
	if err != nil {
		log.Println("can't cache live buses", err)
		return err
	}

	// attempt to parse response to ensure it is valid
	_, _, err = p.Live(agencyID, routeID, "", directionID)
	if err != nil {
		log.Println("can't parse response", err)
		return err
	}

	log.Println("succesfully saved", k)

	return nil
}

func (p mtaNYCBus) Live(agencyID, routeID, stopID string, directionID int) (d []*models.Departure, v []models.Vehicle, err error) {

	stopPointRef := fmt.Sprint("MTA_", stopID)

	u := p.getURL(agencyID, routeID, directionID)

	b, err := etc.RedisGet(u)
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
			v = append(v, models.Vehicle{
				Lat:  act.MonitoredVehicleJourney.VehicleLocation.Latitude,
				Lon:  act.MonitoredVehicleJourney.VehicleLocation.Longitude,
				Live: true,
			})

			for _, oc := range act.MonitoredVehicleJourney.OnwardCalls.OnwardCall {
				if oc.StopPointRef == stopPointRef {
					tripID := act.MonitoredVehicleJourney.FramedVehicleJourneyRef.DatedVehicleJourneyRef
					// remove "MTA NYCT_" from front of string
					if len(tripID) > 9 {
						tripID = tripID[9:]
					}
					if !oc.ExpectedArrivalTime.IsZero() {
						d = append(d, &models.Departure{
							Time:   oc.ExpectedArrivalTime,
							TripID: tripID,
							Live:   true,
						})
					}
				}
			}

		}
	}

	return
}

type call struct {
	ExpectedDepartureTime time.Time
	ExpectedArrivalTime   time.Time

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

type journey struct {
	DirectionRef string
	LineRef      string

	FramedVehicleJourneyRef struct {
		DatedVehicleJourneyRef string
	}

	VehicleLocation struct {
		Latitude  float64
		Longitude float64
	}

	OnwardCalls struct {
		OnwardCall []call
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

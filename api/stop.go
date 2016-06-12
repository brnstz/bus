package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"github.com/julienschmidt/httprouter"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
	"github.com/brnstz/bus/internal/partners"
)

var (
	// stopChan is a channel for receiving requests to get live departure
	// data
	stopChan chan *stopLiveRequest

	// workers is the number of workers processing requestChan concurrently
	stopWorkers = 10
)

type stopLiveRequest struct {
	route    *models.Route
	stop     *models.Stop
	partner  partners.P
	response chan error
}

func init() {
	stopChan = make(chan *stopLiveRequest, 100000)

	for i := 0; i < stopWorkers; i++ {
		go stopWorker()
	}
}

// stop worker calls the partner's live departure API and sets
// req.stop.Live
func stopWorker() {
	for req := range stopChan {
		d, err := req.partner.LiveDepartures(*req.route, *req.stop)

		if err == nil {
			sort.Sort(d)
			for i := 0; i < len(d) && i < models.MaxDepartures; i++ {
				req.stop.Live = append(req.stop.Live, d[i])
			}
		}

		req.response <- err
	}
}

// stopResponse is the value returned by getStops
type stopResponse struct {
	Stops []*models.Stop `json:"stops"`
}

func getStops(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var err error

	lat, err := floatOrDie(r.FormValue("lat"))
	if err != nil {
		return
	}

	lon, err := floatOrDie(r.FormValue("lon"))
	if err != nil {
		return
	}

	meters, err := floatOrDie(r.FormValue("meters"))
	if err != nil {
		return
	}

	filter := r.FormValue("filter")

	sq := models.StopQuery{
		MidLat:     lat,
		MidLon:     lon,
		Dist:       meters,
		RouteType:  filter,
		Departures: true,
		Distinct:   true,
	}
	err = sq.Initialize()
	if err != nil {
		log.Println("can't init stop query", err)
		apiErr(w, err)
		return
	}

	stops, err := models.GetStopsByQuery(etc.DBConn, sq)
	if err != nil {
		log.Println("can't get stops", err)
		apiErr(w, err)
		return
	}

	respch := make(chan error, len(stops))
	count := 0

	for _, s := range stops {
		route, err := models.GetRoute(s.AgencyID, s.RouteID, false)
		if err != nil {
			log.Println("can't get route", err)
			apiErr(w, err)
			return
		}

		// Get a live partner or skip it
		partner, err := partners.Find(*route)
		if err != nil {
			log.Println(err)
			continue
		}

		req := &stopLiveRequest{
			route:    route,
			stop:     s,
			partner:  partner,
			response: respch,
		}

		stopChan <- req
		count++
	}

	for i := 0; i < count; i++ {
		err = <-respch
		if err != nil {
			log.Println(err)
		}
	}

	resp := stopResponse{
		Stops: stops,
	}

	b, err := json.Marshal(resp)
	if err != nil {
		log.Println("can't marshal to json", err)
		apiErr(w, err)
		return
	}

	w.Write(b)
}

package realtime

import (
	"errors"

	"github.com/brnstz/bus/internal/models"
)

var (
	// requestChan is a channel for receiving requests to get live departure
	// data
	requestChan chan *req

	// workers is the number of workers processing requestChan concurrently
	workers = 10

	// ErrNoLiveInfo is returned when there is no way to get live info
	ErrNoLiveInfo = errors.New("there is no live info for this route/stop")
)

// Initialize starts realtime workers
func Initialize() {
	// Create our channel for requests
	requestChan = make(chan *request, 100000)

	// Create workers
	for i := 0; i < workers; i++ {
		go worker()
	}
}

// worker takes a given request and hits its api, sending the response
// on the request's response channel
func worker() {
	for req := range requestChan {
		d, err := req.api.Departures(req.route, req.stop)
		req.Response <- req.Response{
			Departures: d,
			Err:        err,
		}
	}
}

// Request is a single request to add live departures to a route/stop
// combo. The realtime package runs requests concurrently and returns
// the result in Response channel
type Request struct {
	route    models.Route
	stop     models.Stop
	api      R
	Response chan Response
}

// NewRequest creates a request given the route and stop
func NewRequest(route models.Route, stop models.Stop) (*request, error) {
	var api R

	// Typically we have a different API given the agency ID
	switch route.AgencyID {

	case "MTA NYCT":

		// The MTA has a different API for bus and subway
		switch route.Type {

		case models.Subway:
			api = mtaNYCSubway{}

		case models.Bus:
			api = mtaNYCBus{}

		default:
			// No API available
			return nil, ErrNoLiveInfo
		}

	default:
		// No API available
		return nil, ErrNoLiveInfo
	}

	// We successfully got an API, return the request and no error
	return request{
		route:    route,
		stop:     stop,
		api:      api,
		Response: make(chan Response, 1),
	}, nil
}

// Response contains the result of calling the API for this request
type Response struct {
	Departures models.Departures
	Err        error
}

// R is an interface that can pull realtime info (usually from
// a public api) given a stop model
type R interface {
	Departures(models.Route, models.Stop) (models.Departures, error)
	Name() string
}

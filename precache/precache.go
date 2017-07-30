package precache

import (
	"log"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
	"github.com/brnstz/bus/internal/partners"
)

var (

	// max workers per a single agency
	maxWorkersAgency = 5

	// delay is the time to wait between requests for the same route
	delay = time.Duration(60) * time.Second

	// error delay is the time we wait before hitting the API again
	// if we get an error
	errDelay = time.Duration(10) * time.Second

	// size of each precacheRequest's channel
	size = 50000
)

type precacheRequest struct {
	partner     partners.P
	agencyID    string
	routeID     string
	directionID int
	result      chan error
}

// routeWorker runs forever for this partner/agency/route/direction, sending
// a new precacheRequest to the channel ch, delaying between each request.
// The goal is to make a request from the partner before the TTL
// runs out.
func routeWorker(ch chan precacheRequest, p partners.P, agencyID string, routeID string, directionID int) {
	var err error

	// Assume last success was now
	lastSuccess := time.Now()

	// Convert RedisTTL to a duration
	ttlDur := time.Duration(conf.Cache.RedisTTL) * time.Second

	// Loop forever, constantly getting new updates
	for {

		// Create a request
		req := precacheRequest{
			partner:     p,
			agencyID:    agencyID,
			routeID:     routeID,
			directionID: directionID,
			result:      make(chan error, 1),
		}

		// Send it to an agencyWorker
		ch <- req

		// Wait for the response
		err = <-req.result

		// Record current time and difference
		now := time.Now()
		diff := now.Sub(lastSuccess)

		// If the time between successes is greater than cache duration,
		// we should warn that we have a problem
		if diff > ttlDur {
			log.Printf("%v %v %v worker took %v to run, longer than ttl of %v",
				agencyID, routeID, directionID, diff, ttlDur,
			)
		}

		if err != nil {
			log.Println("error getting response", err)
			time.Sleep(errDelay)
		} else {
			lastSuccess = now
			time.Sleep(delay)
		}
	}
}

// agencyWorker calls Precache on each incoming request
func agencyWorker(ch chan precacheRequest) {
	for req := range ch {
		req.result <- req.partner.Precache(req.agencyID, req.routeID, req.directionID)
	}
}

func Precache() {
	// Go through each agency we support
	for _, agencyID := range conf.Partner.AgencyIDs {

		// Create a channel for this agency
		ch := make(chan precacheRequest, size)

		// Create a number of workers for this agency
		for i := 0; i < maxWorkersAgency; i++ {
			go agencyWorker(ch)
		}

		// Get all the routes for this agency
		routes, err := models.GetAllRoutes(etc.DBConn, agencyID)
		if err != nil {
			log.Println("can't get routes", err)
			continue
		}

		for _, route := range routes {

			// Get the partner for each route
			p, err := partners.Find(*route)
			if err == partners.ErrNoPartner {
				// If there's no partner, just ignore it
				continue
			}
			if err != nil {
				// It's a fatal error the precacher if it can't
				// one of its configured partners
				log.Fatal("error getting partner", err)
			}

			for dir := 0; dir <= 1; dir++ {
				// Create a routeWorker for every combination of
				// route / direction. For some APIs, the response for each
				// direction is no different. It's up to the partner to ignore
				// one of the requests.

				// FIXME: problem with this is... we'll never
				// create new goroutines when they are updated in db
				go routeWorker(ch, p, route.AgencyID, route.RouteID, dir)
			}
		}
	}

	// FIXME: wait forever
	select {}
}

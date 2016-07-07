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

	// total simultaneous network connections
	maxWorkers = 200

	// max workers per a single agency
	maxWorkersAgency = 20

	// delay is the time to wait between requests for the same route
	delay = time.Duration(60) * time.Second

	// error delay
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

// routeWorker is
func routeWorker(ch chan precacheRequest, p partners.P, agencyID string, routeID string, directionID int) {
	var err error

	lastSuccess := time.Now()
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

		log.Println("sending req", req)
		// Send it to an agencyWorker
		ch <- req
		log.Println("sent req", req)

		// Wait for the response
		err = <-req.result
		log.Println("got response", req)

		now := time.Now()
		diff := now.Sub(lastSuccess)

		// If the time between successes is greater than cache duration,
		// we should warn that we have a problem
		if diff > ttlDur {
			log.Println("%v %v %v worker took %v to run, longer than ttl of %v",
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
	log.Println("ready for work sarge")
	for req := range ch {
		log.Println("i got one")
		req.result <- req.partner.Precache(req.agencyID, req.routeID, req.directionID)
		log.Println("i did it")
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
			for dir := 0; dir <= 1; dir++ {

				p, err := partners.Find(*route)
				if err == partners.ErrNoPartner {
					continue
				}
				if err != nil {
					log.Fatal("error getting partner", err)
				}

				// Create a routeWorker for every combination
				// FIXME: problem with this is... we'll never
				// create new goroutines when they are updated in db
				go routeWorker(ch, p, route.AgencyID, route.RouteID, dir)
			}
		}
	}

	// FIXME: wait forever
	select {}
}

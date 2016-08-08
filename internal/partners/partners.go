package partners

import (
	"errors"

	"github.com/brnstz/bus/internal/models"
)

var (
	// ErrNoPartner means there is no configured partner for this route
	ErrNoPartner = errors.New("no partner for this route")
)

// P is an interface that can pull live info from partners
type P interface {
	// Precache is called by the precacher binary and saves raw bytes of
	// response into redis. Doing precache prevents clients from hammering
	// partner serves and also ensures our own responses are fast.
	// The precacher will call this function for every valid
	// agency / route / direction combo that returns a partner with
	// Find().
	Precache(agencyID, routeID string, directionID int) error

	// Live reads the data saved into redis by Precache, parses it and
	// returns any Departure and/or Vehicle info that can be appended
	// to the response.
	Live(agencyID, routeID, stopID string, directionID int) ([]*models.Departure, []models.Vehicle, error)
}

// Find returns the correct partner for this route. If there is no
// partner, ErrNoPartner is returned
func Find(route models.Route) (P, error) {

	switch route.AgencyID {

	case "MTA NYCT", "MTABC":
		// The MTA has a different partner API for bus and subway
		switch route.Type {

		case models.Subway, models.Rail:
			_, exists := mtaSubwayRouteToFeed[route.RouteID]
			if exists {
				return mtaNYCSubway{}, nil
			} else {
				return static{}, nil
			}

		case models.Bus:
			return mtaNYCBus{}, nil

		default:
			return static{}, nil
		}

	default:
		return static{}, nil
	}

}

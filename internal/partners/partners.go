package partners

import (
	"errors"
	"log"

	"github.com/brnstz/bus/internal/models"
)

var ErrNoPartner = errors.New("no partner for this route")

// P is an interface that can pull extra info from partners
type P interface {
	Live(models.Route, models.Stop) (models.Departures, []models.Vehicle, error)
}

// Find returns the correct partner for this route. If there is no
// partner, ErrNoPartner is returned
func Find(route models.Route) (P, error) {

	switch route.AgencyID {

	case "MTA NYCT":
		// The MTA has a different partner API for bus and subway
		switch route.Type {

		case models.Subway, models.Rail:
			return mtaNYCSubway{}, nil

		case models.Bus:
			return mtaNYCBus{}, nil

		default:
			log.Println("no partner for", route)
			return nil, ErrNoPartner
		}

	default:
		log.Println("no partner for", route)
		return nil, ErrNoPartner
	}

}

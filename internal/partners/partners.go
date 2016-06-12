package partners

import (
	"errors"

	"github.com/brnstz/bus/internal/models"
)

var ErrNoPartner = errors.New("no partner for this route")

// P is an interface that can pull extra info from partners
type P interface {
	LiveDepartures(models.Route, models.Stop) (models.Departures, error)
}

// Find returns the correct partner for this route. If there is no
// partner, ErrNoPartner is returned
func Find(route models.Route) (P, error) {

	switch route.AgencyID {

	case "MTA NYCT":
		// The MTA has a different partner API for bus and subway
		switch route.Type {

		case models.Subway:
			return mtaNYCSubway{}, nil

		case models.Bus:
			return mtaNYCBus{}, nil

		default:
			return nil, ErrNoPartner
		}

	default:
		return nil, ErrNoPartner
	}

}

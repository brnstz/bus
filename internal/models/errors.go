package models

import "errors"

// This file contains errors we want to specifically identify programmatically

var (
	// ErrNoID is returned by model constructors when no ID is provided
	ErrNoID = errors.New("no ID provided")

	// ErrInvalidRouteType is returned by NewRoute when the route_type
	// cannot be found
	ErrInvalidRouteType = errors.New("invalid route_type")
)

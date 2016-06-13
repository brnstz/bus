package models

// Vehicle is the location (and maybe any other info) we want to provide
// about a
type Vehicle struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`

	// Is this location live or estimated based on scheduled?
	Live bool `json:"live"`
}

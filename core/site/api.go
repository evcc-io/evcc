package site

import "github.com/evcc-io/evcc/core/loadpoint"

// API is the external site API
type API interface {
	Healthy() bool
	LoadPoints() []loadpoint.API
	SetPrioritySoC(float64) error
}

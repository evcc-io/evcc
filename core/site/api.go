package site

import "github.com/evcc-io/evcc/core/loadpoint"

// API is the external site API
type API interface {
	Healthy() bool
	LoadPoints() []loadpoint.API
	SetBufferSoC(float64) error
	SetPrioritySoC(float64) error
	SetResidualPower(float64) error
}

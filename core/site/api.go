package site

import "github.com/evcc-io/evcc/core/loadpoint"

// API is the external site API
type API interface {
	Healthy() bool
	LoadPoints() []loadpoint.API
	SetBufferSoC(float64) error
	GetBufferSoC() float64
	SetPrioritySoC(float64) error
	GetPrioritySoC() float64
	SetResidualPower(float64) error
	GetResidualPower() float64
}

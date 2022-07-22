package site

import "github.com/evcc-io/evcc/core/loadpoint"

// API is the external site API
type API interface {
	Healthy() bool
	LoadPoints() []loadpoint.API
	GetBufferSoC() float64
	SetBufferSoC(float64) error
	GetPrioritySoC() float64
	SetPrioritySoC(float64) error
	GetResidualPower() float64
	SetResidualPower(float64) error
}

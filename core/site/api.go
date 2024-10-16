package site

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
)

// API is the external site API
type API interface {
	Healthy() bool
	Loadpoints() []loadpoint.API
	Vehicles() Vehicles

	// Meta
	GetTitle() string
	SetTitle(string)

	// Config
	GetGridMeterRef() string
	SetGridMeterRef(string)
	GetPVMeterRefs() []string
	SetPVMeterRefs([]string)
	GetBatteryMeterRefs() []string
	SetBatteryMeterRefs([]string)

	// circuits
	GetCircuit() api.Circuit
	SetCircuit(api.Circuit)

	//
	// battery
	//

	GetPrioritySoc() float64
	SetPrioritySoc(float64) error
	GetBufferSoc() float64
	SetBufferSoc(float64) error
	GetBufferStartSoc() float64
	SetBufferStartSoc(float64) error

	// GetBatteryGridChargeLimit get the grid charge limit
	GetBatteryGridChargeLimit() *float64
	// SetBatteryGridChargeLimit sets the grid charge limit
	SetBatteryGridChargeLimit(limit *float64)

	//
	// power and energy
	//

	GetResidualPower() float64
	SetResidualPower(float64) error

	//
	// tariffs and costs
	//

	// GetTariff returns the respective tariff
	GetTariff(string) api.Tariff

	//
	// battery control
	//

	GetBatteryDischargeControl() bool
	SetBatteryDischargeControl(bool) error
}

package site

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/tariff/tariffs"
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

	//
	// battery
	//

	GetBufferSoc() float64
	SetBufferSoc(float64) error
	GetBufferStartSoc() float64
	SetBufferStartSoc(float64) error
	GetPrioritySoc() float64
	SetPrioritySoc(float64) error

	//
	// power and energy
	//

	GetResidualPower() float64
	SetResidualPower(float64) error

	//
	// tariffs and costs
	//

	GetTariffs() tariffs.API
	GetTariff(string) api.Tariff
	GetSmartCostLimit() float64
	SetSmartCostLimit(float64) error

	//
	// battery control
	//

	GetBatteryDischargeControl() bool
	SetBatteryDischargeControl(bool) error
}

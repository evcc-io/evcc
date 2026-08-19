package site

import (
	"iter"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
)

// publisher gives access to the site's publish function
type Publisher interface {
	Publish(key string, val any)
}

// BatteryOptimizerSocGoals are recurring optimizer reserve goals: keep the
// battery at each goal's Soc by its Time on the selected Weekdays. Modelled on
// loadpoint repeating plans (api.RepeatingPlan) so several reserves can be set,
// e.g. an evening reserve plus a morning-peak reserve. Time and Tz are stored
// together so the wall-clock time is always interpreted in its own timezone.

// API is the external site API
type API interface {
	Publisher

	Loadpoints() []loadpoint.API
	ActiveLoadpoints() iter.Seq2[int, loadpoint.API]
	Vehicles() Vehicles
	Optimize()

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
	GetAuxMeterRefs() []string
	SetAuxMeterRefs([]string)
	GetExtMeterRefs() []string
	SetExtMeterRefs([]string)
	GetConsumerMeterRefs() []string
	SetConsumerMeterRefs([]string)

	// circuits
	GetCircuit() api.Circuit

	//
	// battery
	//

	GetBatterySoc() float64
	GetBatteryMaxDischargePower() *float64
	GetPrioritySoc() float64
	SetPrioritySoc(float64) error
	GetBufferSoc() float64
	SetBufferSoc(float64) error
	GetBufferStartSoc() float64
	SetBufferStartSoc(float64) error

	// GetBatteryGridChargeLimit get the grid charge limit
	GetBatteryGridChargeLimit() *float64
	// SetBatteryGridChargeLimit sets the grid charge limit
	SetBatteryGridChargeLimit(limit *float64) error
	GetBatteryOptimizerSocGoals() []api.RepeatingPlan
	SetBatteryOptimizerSocGoals([]api.RepeatingPlan) error

	// GetOptimizerChargingStrategy gets the optimizer grid charging strategy
	GetOptimizerChargingStrategy() string
	// SetOptimizerChargingStrategy sets the optimizer grid charging strategy
	SetOptimizerChargingStrategy(strategy string) error

	//
	// power and energy
	//

	GetGridPower() float64
	GetResidualPower() float64
	SetResidualPower(float64) error
	GetGridExportLimit() float64
	SetGridExportLimit(float64) error

	//
	// tariffs and costs
	//

	// GetTariff returns the respective tariff
	GetTariff(api.TariffUsage) api.Tariff

	//
	// forecast
	//

	// GetSolarAdjusted returns if the solar forecast is adjusted to real production data
	GetSolarAdjusted() bool
	// SetSolarAdjusted sets if the solar forecast is adjusted to real production data
	SetSolarAdjusted(bool)

	//
	// battery control
	//

	GetBatteryDischargeControl() bool
	SetBatteryDischargeControl(bool) error
	GetOptimizerManualPA() *float64
	SetOptimizerManualPA(*float64) error
	GetBatteryGridDischarge() bool
	SetBatteryGridDischarge(bool) error

	//
	// battery control external
	//

	// GetBatteryModeExternal returns the external battery mode
	GetBatteryModeExternal() api.BatteryMode
	// SetBatteryModeExternal sets the external battery mode
	SetBatteryModeExternal(api.BatteryMode) error
}

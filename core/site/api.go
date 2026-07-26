package site

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
)

// publisher gives access to the site's publish function
type Publisher interface {
	Publish(key string, val any)
}

// BatteryOptimizerSocGoal is the daily optimizer reserve goal: keep the battery
// at Soc by Time each day. Time and Tz are stored together so the wall-clock
// time is always interpreted in its own timezone.
type BatteryOptimizerSocGoal struct {
	Soc  float64 `json:"soc"`  // target soc, 1..100
	Time string  `json:"time"` // HH:MM local wall-clock
	Tz   string  `json:"tz"`   // IANA timezone (required)
}

// API is the external site API
type API interface {
	Publisher

	Loadpoints() []loadpoint.API
	Vehicles() Vehicles
	Optimize() error

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
	GetBatteryOptimizerSocGoal() *BatteryOptimizerSocGoal
	SetBatteryOptimizerSocGoal(*BatteryOptimizerSocGoal) error

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

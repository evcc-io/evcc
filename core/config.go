package core

import (
	"time"

	"github.com/andig/evcc/api"
)

// Config contains the public loadpoint configuration
type Config struct {
	Mode api.ChargeMode // Charge mode, guarded by mutex

	// options
	Phases        int64   // Phases- required for converting power and current.
	Voltage       float64 // Operating voltage. 230V for Germany.
	ResidualPower float64 // PV meter only: household usage. Grid meter: household safety margin

	ChargerRef string `mapstructure:"charger"` // Charger reference
	VehicleRef string `mapstructure:"vehicle"` // Vehicle reference

	Meters MetersConfig // Meter references

	Enable, Disable ThresholdConfig
}

// ThresholdConfig defines enable/disable hysteresis parameters
type ThresholdConfig struct {
	Delay     time.Duration
	Threshold float64
}

// MetersConfig contains the loadpoint's meter configuration
type MetersConfig struct {
	GridMeterRef    string   `mapstructure:"grid"`    // Grid usage meter reference
	ChargeMeterRef  string   `mapstructure:"charge"`  // Charger usage meter reference
	PVMeterRef      []string `mapstructure:"pv"`      // PV generation meter reference
	BatteryMeterRef []string `mapstructure:"battery"` // Battery charging meter reference
}

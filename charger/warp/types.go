package warp

const (
	FeatureMeter          = "meter"
	FeatureMeterAllValues = "meter_all_values"
	FeatureMeterPhases    = "meter_phases"
	FeatureNfc            = "nfc"
	FeatureMeters         = "meters"
	FeaturePhaseSwitch    = "phase_switch"
)

// https://www.warp-charger.com/api.html#evse_state
type EvseState struct {
	Iec61851State int `json:"iec61851_state"`
}

type EvseExternalCurrent struct {
	Current int `json:"current"`
}

type EvseUserEnabled struct {
	Enabled bool `json:"enabled"`
}

type Evse struct {
	State           EvseState
	ExternalCurrent EvseExternalCurrent
	UserCurrent     EvseExternalCurrent
	UserEnabled     EvseUserEnabled
}

type MeterValues struct {
	Power     float64 `json:"power"`
	EnergyRel float64 `json:"energy_rel"`
	EnergyAbs float64 `json:"energy_abs"`
	Currents  [3]float64
	Voltages  [3]float64
	Values    []float64
}

// Meter value IDs according to Tinkerforge meter_value_id.csv
const (
	ValueIDVoltageL1N       = 1   // Voltage L1-N
	ValueIDVoltageL2N       = 2   // Voltage L2-N
	ValueIDVoltageL3N       = 3   // Voltage L3-N
	ValueIDCurrentImExSumL1 = 13  // Current L1 Im-Ex Sum
	ValueIDCurrentImExSumL2 = 17  // Current L2 Im-Ex Sum
	ValueIDCurrentImExSumL3 = 21  // Current L3 Im-Ex Sum
	ValueIDPowerImExSum     = 74  // Power Im-Ex Sum L1 L2 L3
	ValueIDEnergyAbsImSum   = 209 // Energy Im Sum L1 L2 L3
)

type ChargeTrackerCurrentCharge struct {
	AuthorizationInfo struct {
		TagType int    `json:"tag_type"`
		TagId   string `json:"tag_id"`
	} `json:"authorization_info"`
}

//go:generate go tool enumer -type ExternalControl -trimprefix ExternalControl -transform whitespace
type ExternalControl int

const (
	ExternalControlAvailable ExternalControl = iota
	ExternalControlDeactivated
	ExternalControlRuntimeConditionsNotMet
	ExternalControlCurrentlySwitching
)

type PmState struct {
	ExternalControl ExternalControl `json:"external_control"`
}

type PmLowLevelState struct {
	Is3phase bool `json:"is_3phase"`
}

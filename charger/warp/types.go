package warp

const (
	FeatureMeter          = "meter"
	FeatureMeters         = "meters"
	FeatureMeterAllValues = "meter_all_values"
	FeatureMeterPhases    = "meter_phases"
	FeatureNfc            = "nfc"
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
	TmpValues []float64
}

type PhasePair struct {
	CurrentID int
	VoltageID int
}

type MeterSchema struct {
	PowerID     int
	EnergyAbsID int
	Phases      [3]PhasePair
}

// Value IDs based on Tinkerforge's meter_value_id.csv
var DefaultSchema = MeterSchema{
	PowerID:     74,  // Power Im-Ex Sum L1 L2 L3
	EnergyAbsID: 209, // Energy Im Sum L1 L2 L3
	Phases: [3]PhasePair{
		{CurrentID: 13, VoltageID: 1}, // Current L1 Im-Ex Sum, Voltage L1-N
		{CurrentID: 17, VoltageID: 2}, // Current L2 Im-Ex Sum, Voltage L2-N
		{CurrentID: 21, VoltageID: 3}, // Current L3 Im-Ex Sum, Voltage L3-N
	},
}

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

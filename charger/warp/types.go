package warp

const (
	FeatureIso15118       = "iso15118"
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

type EvsePhaseAutoSwitch struct {
	Enabled bool `json:"enabled"`
}

type EvseLowLevelState struct {
	Uptime uint32 `json:"uptime"`
}

type EvseSlot struct {
	MaxCurrent int32 `json:"max_current"`
}

type Evse struct {
	State           EvseState
	Slots           []EvseSlot
	ExternalCurrent EvseExternalCurrent
	UserCurrent     EvseExternalCurrent
	UserEnabled     EvseUserEnabled
	LowLevelState   EvseLowLevelState
}

type MeterValues struct {
	Power     float64 `json:"power"`
	EnergyRel float64 `json:"energy_rel"`
	EnergyAbs float64 `json:"energy_abs"`
	Currents  [3]float64
	Voltages  [3]float64
}

//go:generate go tool enumer -type Mvid -trimprefix Mvid -transform whitespace
type Mvid int

// See https://github.com/Tinkerforge/esp32-firmware/blob/master/software/src/modules/meters/meter_value_id.csv
const (
	MvidPower     Mvid = 74
	MvidEnergy    Mvid = 209
	MvidCurrentL1 Mvid = 13
	MvidCurrentL2 Mvid = 17
	MvidCurrentL3 Mvid = 21
	MvidVoltageL1 Mvid = 1
	MvidVoltageL2 Mvid = 2
	MvidVoltageL3 Mvid = 3
	MvidPowerL1   Mvid = 39
	MvidPowerL2   Mvid = 48
	MvidPowerL3   Mvid = 57
)

type Name struct {
	Name        string `json:"name"`
	WarpType    string `json:"type"`
	DisplayType string `json:"display_type"`
	Uid         string `json:"uid"`
}

// WARP4 only: vehicle data read via ISO 15118; soc is null and mac is empty while unknown
type EvState struct {
	Soc      *float64 `json:"soc"`
	Mac      string   `json:"mac"`
	Capacity *float64 `json:"capacity"`
}

type FloatWithNaN float64

type ChargeTrackerCurrentCharge struct {
	UserId            int          `json:"user_id"`
	EvseUptimeStart   uint32       `json:"evse_uptime_start"`
	TimestampMinutes  int          `json:"timestamp_minutes"`
	MeterStart        FloatWithNaN `json:"meter_start"`
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

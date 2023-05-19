package warp

const (
	FeatureMeter       = "meter"
	FeatureMeterPhases = "meter_phases"
	FeatureNfc         = "nfc"
)

// https://www.warp-charger.com/api.html#evse_state
type EvseState struct {
	Iec61851State          int   `json:"iec61851_state"`
	ChargerState           int   `json:"charger_state"`
	ContactorState         int   `json:"contactor_state"`
	ContactorError         int   `json:"contactor_error"`
	AllowedChargingCurrent int64 `json:"allowed_charging_current"`
	ErrorState             int   `json:"error_state"`
	LockState              int   `json:"lock_state"`
}

type EvseExternalCurrent struct {
	Current int `json:"current"`
}

// https://www.warp-charger.com/api.html#evse_low_level_state
type LowLevelState struct {
	TimeSinceStateChange int64 `json:"time_since_state_change"`
	Uptime               int64 `json:"uptime"`
	LedState             int   `json:"led_state"`
	CpPwmDutyCycle       int   `json:"cp_pwm_duty_cycle"`
	AdcValues            []int `json:"adc_values"`
	Voltages             []int
	Resistances          []int
	Gpio                 []bool
}

// https://www.warp-charger.com/api.html#meter_state
type MeterState struct {
	State int `json:"state"` // Warp 1 only
}

// https://www.warp-charger.com/api.html#meter_values
type MeterValues struct {
	Power           float64 `json:"power"`
	EnergyRel       float64 `json:"energy_rel"`
	EnergyAbs       float64 `json:"energy_abs"`
	PhasesActive    []bool  `json:"phases_active"`
	PhasesConnected []bool  `json:"phases_connected"`
}

// https://www.warp-charger.com/api.html#meter_all_values
type MeterAllValues struct {
	PhasesActive    []bool `json:"phases_active"`
	PhasesConnected []bool `json:"phases_connected"`
}

type UsersConfig struct {
	Users []User `json:"users"`
}

type ChargeTrackerCurrentCharge struct {
	UserID            int     `json:"user_id"`
	MeterStart        float64 `json:"meter_start"`
	AuthorizationType int     `json:"authorization_type"`
	AuthorizationInfo struct {
		TagType int    `json:"tag_type"`
		TagId   string `json:"tag_id"`
	} `json:"authorization_info"`
}

type User struct {
	ID          int    `json:"id"`
	Roles       int    `json:"roles"`
	Current     int    `json:"current"`
	DisplayName string `json:"display_name"`
	UserName    string `json:"username"`
}

type LastNfcTag struct {
	UserID int    `json:"user_id"`
	Type   int    `json:"tag_type"`
	ID     string `json:"tag_id"`
}

type EmConfig struct {
	ContactorInstalled bool `json:"contactor_installed"`
	PhaseSwitchingMode int  `json:"phase_switching_mode"`
}

//go:generate enumer -type ExternalControl -trimprefix ExternalControl -transform whitespace
type ExternalControl int

const (
	ExternalControlAvailable ExternalControl = iota
	ExternalControlDeactivated
	ExternalControlRuntimeConditionsNotMet
	ExternalControlCurrentlySwitching
)

type EmState struct {
	ExternalControl ExternalControl `json:"external_control"`
	PhasesSwitched  int             `json:"phases_switched"`
	Input3State     bool            `json:"input3_state"`
	Input4State     bool            `json:"input4_state"`
	RelayState      bool            `json:"relay_state"`
	ErrorFlags      int             `json:"error_flags"`
}

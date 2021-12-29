package warp

import "time"

const (
	RootTopic = "warp"
	Timeout   = 30 * time.Second
)

type Status struct {
	Iec61851State          int   `json:"iec61851_state"`
	VehicleState           int   `json:"vehicle_state"`
	ChargeRelease          int   `json:"charge_release"`
	ContactorState         int   `json:"contactor_state"`
	ContactorError         int   `json:"contactor_error"`
	AllowedChargingCurrent int64 `json:"allowed_charging_current"`
	ErrorState             int   `json:"error_state"`
	LockState              int   `json:"lock_state"`
	TimeSinceStateChange   int64 `json:"time_since_state_change"`
	Uptime                 int64 `json:"uptime"`
}

type LowLevelState struct {
	LedState       int   `json:"led_state"`
	CpPwmDutyCycle int   `json:"cp_pwm_duty_cycle"`
	AdcValues      []int `json:"adc_values"`
	Voltages       []int
	Resistances    []int
	Gpio           []bool
}

type PowerStatus struct {
	Power     float64 `json:"power"`
	EnergyRel float64 `json:"energy_rel"`
	EnergyAbs float64 `json:"energy_abs"`
}

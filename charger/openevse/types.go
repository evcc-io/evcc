package openevse

const (
	Enabled  = "active"
	Disabled = "disabled"
)

// Only keep the interesting properties from the status endpoint
type Status struct {
	Amp            *float64 `json:"amp,omitempty"`             // the value of the charge current in mA
	Elapsed        *float64 `json:"elapsed,omitempty"`         // current session duration in seconds
	ManualOverride *int     `json:"manual_override,omitempty"` // 1 = active, 0 = default
	Mode           *string  `json:"mode,omitempty"`            // The current mode of the EVSE
	Pilot          *int     `json:"pilot,omitempty"`           // the pilot value, in amps
	Power          *float64 `json:"power,omitempty"`           // apparent power in watts
	SessionEnergy  *float64 `json:"session_energy"`            // The total amount of energy accumulated for current session (in wh)
	SessionElapsed *int     `json:"session_elapsed"`           // duration of this charging session in seconds
	State          *int     `json:"state,omitempty"`           // evse state 1=A 2=B 3=C 4=D 5-11=F 254=sleeping 255=disabled
	Status         *string  `json:"status,omitempty"`          // active, disabled, none, unknown
	TotalEnergy    *float64 `json:"total_energy"`              // The total amount of energy accumulated (in kwh)
	Vehicle        *int     `json:"vehicle,omitempty"`         // 0=not connected, 1=connected
	Voltage        *float64 `json:"voltage,omitempty"`         // supplied via MQTT/Tesla/HTTP or assume a default
}

type Override struct {
	State         *string `json:"state,omitempty"`          // Either enable charging (active) or block charging (disabled)
	ChargeCurrent *int    `json:"charge_current,omitempty"` // Specify the active charge current in Amps >= 0
	MaxCurrent    *int    `json:"max_current,omitempty"`    // Maximum current, primarily for load sharing situations
	AutoRelease   *bool   `json:"auto_release,omitempty"`
}

package ghostone

// Relais switch state values
const (
	RelaisStateOnePhase         = "onePhase"
	RelaisStateThreePhase       = "threePhase"
	RelaisStateNotAvailable     = "notAvailable"
	RelaisStateSwitchInProgress = "switchInProgress"
	RelaisStateNotPossible      = "notPossible"
)

// PV optimization mode values
const (
	PvModeNone = "noPV"
)

// Enabled is the response from GET /system/relais-switch/enabled
type Enabled struct {
	Enabled bool `json:"enabled"`
}

// RelaisSwitchStateRead is the response from GET /system/relais-switch/state
type RelaisSwitchStateRead struct {
	Value            string `json:"value"`            // onePhase, threePhase, notAvailable, switchInProgress, notPossible
	Mode             string `json:"mode"`             // manual, automatic
	LimitationReason string `json:"limitationReason"` // plcLimitation, hlcLimitation, gridLimitation
	CurrentState     string `json:"currentState"`     // onePhase, threePhase
}

// RelaisSwitchStateWrite is the request body for PUT /system/relais-switch/state
type RelaisSwitchStateWrite struct {
	Value string `json:"value"` // onePhase, threePhase, automatic
}

// PvOptimizationMode is the response from GET /charging/pvoptimization/mode
type PvOptimizationMode struct {
	Value string `json:"value"` // noPV, pvOnly, pvPlus
}

// RfidCardLastRead is the response from GET /rfid-cards/last-read
type RfidCardLastRead struct {
	UUID string `json:"uuid"`
	ID   string `json:"id,omitempty"`
}

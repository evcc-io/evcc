package easee

// API is the Easee API endpoint
const API = "https://api.easee.com/api"

const (
	ChargeStart  = "start_charging"
	ChargeStop   = "stop_charging"
	ChargePause  = "pause_charging"
	ChargeResume = "resume_charging"
)

// charge mode definition
const (
	ModeOffline                int = 0
	ModeDisconnected           int = 1
	ModeAwaitingStart          int = 2
	ModeCharging               int = 3
	ModeCompleted              int = 4
	ModeError                  int = 5
	ModeReadyToCharge          int = 6
	ModeAwaitingAuthentication int = 7
	ModeDeauthenticating       int = 8
)

// Charger is the charger type
type Charger struct {
	ID   string
	Name string
}

type ChargerConfig struct {
	IsEnabled                    bool
	LockCablePermanently         bool
	AuthorizationRequired        bool
	RemoteStartRequired          bool
	SmartButtonEnabled           bool
	WiFiSSID                     string
	DetectedPowerGridType        int
	OfflineChargingMode          int
	CircuitMaxCurrentP1          float64
	CircuitMaxCurrentP2          float64
	CircuitMaxCurrentP3          float64
	EnableIdleCurrent            bool
	LimitToSinglePhaseCharging   bool
	PhaseMode                    int
	LocalNodeType                int
	LocalAuthorizationRequired   bool
	LocalRadioChannel            int
	LocalShortAddress            int
	LocalParentAddrOrNumOfNodes  int
	LocalPreAuthorizeEnabled     bool
	LocalAuthorizeOfflineEnabled bool
	AllowOfflineTxForUnknownId   bool
	MaxChargerCurrent            float64
	LedStripBrightness           int
}

// Site is the site type
type Site struct {
	ID       int
	SiteKey  string
	Name     string
	Circuits []Circuit
}

// Circuit is the circuit type
type Circuit struct {
	ID               int
	SiteID           int
	CircuitPanelID   int
	PanelName        string
	RatedCurrent     float64
	UseDynamicMaster bool
	ParentCircuitID  int
	Chargers         []Charger
}

// ChargerStatus is the charger status type
type ChargerStatus struct {
	SmartCharging                                bool
	CableLocked                                  bool
	ChargerOpMode                                int
	TotalPower                                   float64
	SessionEnergy                                float64
	EnergyPerHour                                float64
	WiFiRSSI                                     int
	CellRSSI                                     int
	LocalRSSI                                    int
	OutputPhase                                  int
	DynamicCircuitCurrentP1                      float64
	DynamicCircuitCurrentP2                      float64
	DynamicCircuitCurrentP3                      float64
	LatestPulse                                  string
	ChargerFirmware                              int
	LatestFirmware                               int
	Voltage                                      float64
	ChargerRAT                                   int
	LockCablePermanently                         bool
	InCurrentT2                                  float64
	InCurrentT3                                  float64
	InCurrentT4                                  float64
	InCurrentT5                                  float64
	OutputCurrent                                float64
	IsOnline                                     bool
	InVoltageT1T2                                float64
	InVoltageT1T3                                float64
	InVoltageT1T4                                float64
	InVoltageT1T5                                float64
	InVoltageT2T3                                float64
	InVoltageT2T4                                float64
	InVoltageT2T5                                float64
	InVoltageT3T4                                float64
	InVoltageT3T5                                float64
	InVoltageT4T5                                float64
	LedMode                                      int
	CableRating                                  float64
	DynamicChargerCurrent                        float64
	CircuitTotalAllocatedPhaseConductorCurrentL1 float64
	CircuitTotalAllocatedPhaseConductorCurrentL2 float64
	CircuitTotalAllocatedPhaseConductorCurrentL3 float64
	CircuitTotalPhaseConductorCurrentL1          float64
	CircuitTotalPhaseConductorCurrentL2          float64
	CircuitTotalPhaseConductorCurrentL3          float64
	ReasonForNoCurrent                           int
	WiFiAPEnabled                                bool
	LifetimeEnergy                               float64
	OfflineMaxCircuitCurrentP1                   int
	OfflineMaxCircuitCurrentP2                   int
	OfflineMaxCircuitCurrentP3                   int
}

// ChargerSettings is the charger settings type
type ChargerSettings struct {
	Enabled                      *bool    `json:"enabled,omitempty"`
	EnableIdleCurrent            *bool    `json:"enableIdleCurrent,omitempty"`
	LimitToSinglePhaseCharging   *bool    `json:"limitToSinglePhaseCharging,omitempty"`
	LockCablePermanently         *bool    `json:"lockCablePermanently,omitempty"`
	SmartButtonEnabled           *bool    `json:"smartButtonEnabled,omitempty"`
	PhaseMode                    *int     `json:"phaseMode,omitempty"`
	SmartCharging                *bool    `json:"smartCharging,omitempty"`
	LocalPreAuthorizeEnabled     *bool    `json:"localPreAuthorizeEnabled,omitempty"`
	LocalAuthorizeOfflineEnabled *bool    `json:"localAuthorizeOfflineEnabled,omitempty"`
	AllowOfflineTxForUnknownID   *bool    `json:"allowOfflineTxForUnknownId,omitempty"`
	OfflineChargingMode          *int     `json:"offlineChargingMode,omitempty"`
	AuthorizationRequired        *bool    `json:"authorizationRequired,omitempty"`
	RemoteStartRequired          *bool    `json:"remoteStartRequired,omitempty"`
	LedStripBrightness           *int     `json:"ledStripBrightness,omitempty"`
	MaxChargerCurrent            *int     `json:"maxChargerCurrent,omitempty"`
	DynamicChargerCurrent        *float64 `json:"dynamicChargerCurrent,omitempty"`
}

// CircuitSettings is the circuit settings type
type CircuitSettings struct {
	DynamicCircuitCurrentP1    *float64 `json:"dynamicCircuitCurrentP1,omitempty"`
	DynamicCircuitCurrentP2    *float64 `json:"dynamicCircuitCurrentP2,omitempty"`
	DynamicCircuitCurrentP3    *float64 `json:"dynamicCircuitCurrentP3,omitempty"`
	MaxCircuitCurrentP1        *float64 `json:"maxCircuitCurrentP1,omitempty"`
	MaxCircuitCurrentP2        *float64 `json:"maxCircuitCurrentP2,omitempty"`
	MaxCircuitCurrentP3        *float64 `json:"maxCircuitCurrentP3,omitempty"`
	EnableIdleCurrent          *bool    `json:"enableIdleCurrent,omitempty"`
	OfflineMaxCircuitCurrentP1 *int     `json:"offlineMaxCircuitCurrentP1,omitempty"`
	OfflineMaxCircuitCurrentP2 *int     `json:"offlineMaxCircuitCurrentP2,omitempty"`
	OfflineMaxCircuitCurrentP3 *int     `json:"offlineMaxCircuitCurrentP3,omitempty"`
}

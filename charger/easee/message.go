package easee

// API is the Easee API endpoint
const API = "https://api.easee.cloud/api"

// charge mode definition
const (
	ModeOffline       int = 0
	ModeDisconnected  int = 1
	ModeAwaitingStart int = 2
	ModeCharging      int = 3
	ModeCompleted     int = 4
	ModeError         int = 5
	ModeReadyToCharge int = 6
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
}

// ChargerSettings is the charger settings type
type ChargerSettings struct {
	Enabled                      *bool `json:"enabled,omitempty"`
	EnableIdleCurrent            *bool `json:"enableIdleCurrent,omitempty"`
	LimitToSinglePhaseCharging   *bool `json:"limitToSinglePhaseCharging,omitempty"`
	LockCablePermanently         *bool `json:"lockCablePermanently,omitempty"`
	SmartButtonEnabled           *bool `json:"smartButtonEnabled,omitempty"`
	PhaseMode                    *int  `json:"phaseMode,omitempty"`
	SmartCharging                *bool `json:"smartCharging,omitempty"`
	LocalPreAuthorizeEnabled     *bool `json:"localPreAuthorizeEnabled,omitempty"`
	LocalAuthorizeOfflineEnabled *bool `json:"localAuthorizeOfflineEnabled,omitempty"`
	AllowOfflineTxForUnknownID   *bool `json:"allowOfflineTxForUnknownId,omitempty"`
	OfflineChargingMode          *int  `json:"offlineChargingMode,omitempty"`
	AuthorizationRequired        *bool `json:"authorizationRequired,omitempty"`
	RemoteStartRequired          *bool `json:"remoteStartRequired,omitempty"`
	LedStripBrightness           *int  `json:"ledStripBrightness,omitempty"`
	MaxChargerCurrent            *int  `json:"maxChargerCurrent,omitempty"`
	DynamicChargerCurrent        *int  `json:"dynamicChargerCurrent,omitempty"`
}

// CircuitSettings is the circuit settings type
type CircuitSettings struct {
	DynamicCircuitCurrentP1    *float64 `json:"dynamicCircuitCurrentP1,omitempty"`
	DynamicCircuitCurrentP2    *float64 `json:"dynamicCircuitCurrentP2,omitempty"`
	DynamicCircuitCurrentP3    *float64 `json:"dynamicCircuitCurrentP3,omitempty"`
	MaxCircuitCurrentP1        *int     `json:"maxCircuitCurrentP1,omitempty"`
	MaxCircuitCurrentP2        *int     `json:"maxCircuitCurrentP2,omitempty"`
	MaxCircuitCurrentP3        *int     `json:"maxCircuitCurrentP3,omitempty"`
	EnableIdleCurrent          *bool    `json:"enableIdleCurrent,omitempty"`
	OfflineMaxCircuitCurrentP1 *int     `json:"offlineMaxCircuitCurrentP1,omitempty"`
	OfflineMaxCircuitCurrentP2 *int     `json:"offlineMaxCircuitCurrentP2,omitempty"`
	OfflineMaxCircuitCurrentP3 *int     `json:"offlineMaxCircuitCurrentP3,omitempty"`
}

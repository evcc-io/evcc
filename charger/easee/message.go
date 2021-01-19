package easee

// Charger is the charger type
type Charger struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Site is the site type
type Site struct {
	ID       int       `json:"id"`
	SiteKey  string    `json:"siteKey"`
	Name     string    `json:"name"`
	Circuits []Circuit `json:"circuits"`
}

// Circuit is the circuit type
type Circuit struct {
	ID               int     `json:"id"`
	SiteID           int     `json:"siteId"`
	CircuitPanelID   int     `json:"circuitPanelId"`
	PanelName        string  `json:"panelName"`
	RatedCurrent     float32 `json:"ratedCurrent"`
	UseDynamicMaster bool    `json:"useDynamicMaster"`
	ParentCircuitID  int     `json:"parentCircuitId"`
	// "chargers": [{
	// 	"id": "string",
	// 	"name": "string",
	// 	"color": 1,
	// 	"createdOn": "2021-01-19T12:39:10.359Z",
	// 	"updatedOn": "2021-01-19T12:39:10.359Z",
	// 	"backPlate": {
	// 		"id": "string",
	// 		"masterBackPlateId": "string",
	// 		"name": "string"
	// 	},
	// 	"levelOfAccess": 1,
	// 	"productCode": 1,
	// 	"userRole": 1,
	// 	"isTemporary": true
	// }],
	// "masterBackplate": {
	// 	"id": "string",
	// 	"masterBackPlateId": "string",
	// 	"name": "string"
	// },
}

// ChargerStatus is the charger status type
type ChargerStatus struct {
	SmartCharging                                bool    `json:"smartCharging"`
	CableLocked                                  bool    `json:"cableLocked"`
	ChargerOpMode                                int     `json:"chargerOpMode"`
	TotalPower                                   float32 `json:"totalPower"`
	SessionEnergy                                float32 `json:"sessionEnergy"`
	EnergyPerHour                                float32 `json:"energyPerHour"`
	WiFiRSSI                                     int     `json:"wiFiRSSI"`
	CellRSSI                                     int     `json:"cellRSSI"`
	LocalRSSI                                    int     `json:"localRSSI"`
	OutputPhase                                  int     `json:"outputPhase"`
	DynamicCircuitCurrentP1                      float32 `json:"dynamicCircuitCurrentP1"`
	DynamicCircuitCurrentP2                      float32 `json:"dynamicCircuitCurrentP2"`
	DynamicCircuitCurrentP3                      float32 `json:"dynamicCircuitCurrentP3"`
	LatestPulse                                  string  `json:"latestPulse"`
	ChargerFirmware                              int     `json:"chargerFirmware"`
	LatestFirmware                               int     `json:"latestFirmware"`
	Voltage                                      float32 `json:"voltage"`
	ChargerRAT                                   int     `json:"chargerRAT"`
	LockCablePermanently                         bool    `json:"lockCablePermanently"`
	InCurrentT2                                  float32 `json:"inCurrentT2"`
	InCurrentT3                                  float32 `json:"inCurrentT3"`
	InCurrentT4                                  float32 `json:"inCurrentT4"`
	InCurrentT5                                  float32 `json:"inCurrentT5"`
	OutputCurrent                                float32 `json:"outputCurrent"`
	IsOnline                                     bool    `json:"isOnline"`
	InVoltageT1T2                                float32 `json:"inVoltageT1T2"`
	InVoltageT1T3                                float32 `json:"inVoltageT1T3"`
	InVoltageT1T4                                float32 `json:"inVoltageT1T4"`
	InVoltageT1T5                                float32 `json:"inVoltageT1T5"`
	InVoltageT2T3                                float32 `json:"inVoltageT2T3"`
	InVoltageT2T4                                float32 `json:"inVoltageT2T4"`
	InVoltageT2T5                                float32 `json:"inVoltageT2T5"`
	InVoltageT3T4                                float32 `json:"inVoltageT3T4"`
	InVoltageT3T5                                float32 `json:"inVoltageT3T5"`
	InVoltageT4T5                                float32 `json:"inVoltageT4T5"`
	LedMode                                      int     `json:"ledMode"`
	CableRating                                  float32 `json:"cableRating"`
	DynamicChargerCurrent                        float32 `json:"dynamicChargerCurrent"`
	CircuitTotalAllocatedPhaseConductorCurrentL1 float32 `json:"circuitTotalAllocatedPhaseConductorCurrentL1"`
	CircuitTotalAllocatedPhaseConductorCurrentL2 float32 `json:"circuitTotalAllocatedPhaseConductorCurrentL2"`
	CircuitTotalAllocatedPhaseConductorCurrentL3 float32 `json:"circuitTotalAllocatedPhaseConductorCurrentL3"`
	CircuitTotalPhaseConductorCurrentL1          float32 `json:"circuitTotalPhaseConductorCurrentL1"`
	CircuitTotalPhaseConductorCurrentL2          float32 `json:"circuitTotalPhaseConductorCurrentL2"`
	CircuitTotalPhaseConductorCurrentL3          float32 `json:"circuitTotalPhaseConductorCurrentL3"`
	ReasonForNoCurrent                           int     `json:"reasonForNoCurrent"`
	WiFiAPEnabled                                bool    `json:"wiFiAPEnabled"`
	LifetimeEnergy                               float32 `json:"lifetimeEnergy"`
	OfflineMaxCircuitCurrentP1                   int     `json:"offlineMaxCircuitCurrentP1"`
	OfflineMaxCircuitCurrentP2                   int     `json:"offlineMaxCircuitCurrentP2"`
	OfflineMaxCircuitCurrentP3                   int     `json:"offlineMaxCircuitCurrentP3"`
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
	DynamicCircuitCurrentP1    *int  `json:"dynamicCircuitCurrentP1,omitempty"`
	DynamicCircuitCurrentP2    *int  `json:"dynamicCircuitCurrentP2,omitempty"`
	DynamicCircuitCurrentP3    *int  `json:"dynamicCircuitCurrentP3,omitempty"`
	MaxCircuitCurrentP1        *int  `json:"maxCircuitCurrentP1,omitempty"`
	MaxCircuitCurrentP2        *int  `json:"maxCircuitCurrentP2,omitempty"`
	MaxCircuitCurrentP3        *int  `json:"maxCircuitCurrentP3,omitempty"`
	EnableIdleCurrent          *bool `json:"enableIdleCurrent,omitempty"`
	OfflineMaxCircuitCurrentP1 *int  `json:"offlineMaxCircuitCurrentP1,omitempty"`
	OfflineMaxCircuitCurrentP2 *int  `json:"offlineMaxCircuitCurrentP2,omitempty"`
	OfflineMaxCircuitCurrentP3 *int  `json:"offlineMaxCircuitCurrentP3,omitempty"`
}

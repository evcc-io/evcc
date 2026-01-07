package ecoflow

// ecoflowResponse is a generic response wrapper for EcoFlow API responses
type ecoflowResponse[T any] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// EcoFlowStreamData represents the full device status from EcoFlow Stream API
// https://developer-eu.ecoflow.com/us/document/bkw
type EcoFlowStreamData struct {
	Relay2Onoff               bool            `json:"relay2Onoff"`         // AC1 switch (false=off, true=on)
	Relay3Onoff               bool            `json:"relay3Onoff"`         // AC2 switch (false=off, true=on)
	PowGetPvSum               float64         `json:"powGetPvSum"`         // Real-time PV power (W)
	FeedGridMode              int             `json:"feedGridMode"`        // Feed-in control (1-off, 2-on)
	GridConnectionPower       float64         `json:"gridConnectionPower"` // Grid port power (W)
	PowGetSysGrid             float64         `json:"powGetSysGrid"`       // System real-time grid power (W)
	PowGetSysLoad             float64         `json:"powGetSysLoad"`       // System real-time load power (W)
	CmsBattSoc                float64         `json:"cmsBattSoc"`          // Battery SOC (%)
	PowGetBpCms               float64         `json:"powGetBpCms"`         // Real-time battery power (W)
	BackupReverseSoc          int             `json:"backupReverseSoc"`    // Backup reserve level (%)
	CmsMaxChgSoc              int             `json:"cmsMaxChgSoc"`        // Charge limit (%)
	CmsMinDsgSoc              int             `json:"cmsMinDsgSoc"`        // Discharge limit (%)
	EnergyStrategyOperateMode map[string]bool `json:"energyStrategyOperateMode"`
	QuotaCloudTs              string          `json:"quota_cloud_ts"`
}

// EcoFlowPowerStreamData represents the full device status from PowerStream API
// https://developer-eu.ecoflow.com/us/document/wn511
type EcoFlowPowerStreamData struct {
	// Power values (from heartbeat)
	Pv1InputWatts  float64 `json:"pv1InputWatts"`  // PV1 input power (W)
	Pv2InputWatts  float64 `json:"pv2InputWatts"`  // PV2 input power (W)
	BatInputWatts  float64 `json:"batInputWatts"`  // Battery input/output power (W), positive=discharge, negative=charge
	BatInputCur    int     `json:"batInputCur"`    // Battery current (0.1A), positive=discharge, negative=charge
	InvOutputWatts float64 `json:"invOutputWatts"` // Inverter AC output power (W)
	BatSoc         int     `json:"batSoc"`         // Battery SOC (%)
	SupplyPriority int     `json:"supplyPriority"` // Power supply priority (0=supply, 1=storage)
	InvOnOff       int     `json:"invOnOff"`       // Inverter on/off (0=off, 1=on)
	PermanentWatts int     `json:"permanentWatts"` // Custom load power (W)
	LowerLimit     int     `json:"lowerLimit"`     // Battery discharge limit (%)
	UpperLimit     int     `json:"upperLimit"`     // Battery charge limit (%)
	FeedProtect    int     `json:"feedProtect"`    // Feed-in protection (0=off, 1=on)
}

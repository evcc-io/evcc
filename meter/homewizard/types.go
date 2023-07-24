package homewizard

// StateResponse returns the actual state of the HomeWizard Energy Socket
// https://homewizard-energy-api.readthedocs.io/endpoints.html#state-api-v1-state
type StateResponse struct {
	PowerOn    bool `json:"power_on"`
	SwitchLock bool `json:"switch_lock"`
	Brightness int  `json:"brightness"`
}

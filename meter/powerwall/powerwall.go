package powerwall

// credits to https://github.com/vloschiavo/powerwall2

// URIs
const (
	MeterURI   = "/api/meters/aggregates"
	BatteryURI = "/api/system_status/soe"
	LoginURI   = "/api/login/Basic"
)

// MeterResponse is the /api/system_status/aggregates response
type MeterResponse map[string]struct {
	LastCommunicationTime string  `json:"last_communication_time"`
	InstantPower          float64 `json:"instant_power"`
	InstantReactivePower  float64 `json:"instant_reactive_power"`
	InstantApparentPower  float64 `json:"instant_apparent_power"`
	Frequency             float64 `json:"frequency"`
	EnergyExported        float64 `json:"energy_exported"`
	EnergyImported        float64 `json:"energy_imported"`
	InstantAverageVoltage float64 `json:"instant_average_voltage"`
	InstantTotalCurrent   float64 `json:"instant_total_current"`
	IACurrent             float64 `json:"i_a_current"`
	IBCurrent             float64 `json:"i_b_current"`
	ICCurrent             float64 `json:"i_c_current"`
}

// BatteryResponse is the /api/system_status/soe response
type BatteryResponse struct {
	Percentage float64 `json:"percentage"`
}

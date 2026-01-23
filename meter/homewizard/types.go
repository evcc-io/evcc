package homewizard

// ApiResponse returns allows you to get basic information from the HomeWizard Energy Socket
// https://homewizard-energy-api.readthedocs.io/endpoints.html#basic-information-api
type ApiResponse struct {
	ProductType string `json:"product_type"`
	ApiVersion  string `json:"api_version"`
}

// StateResponse returns the actual state of the HomeWizard Energy Socket
// https://homewizard-energy-api.readthedocs.io/endpoints.html#recent-measurement-api-v1-data
type StateResponse struct {
	PowerOn bool `json:"power_on"`
}

// DataResponse returns the most recent measurements from the HomeWizard device
// https://homewizard-energy-api.readthedocs.io/endpoints.html#state-api-v1-state
type DataResponse struct {
	ActivePowerW        float64 `json:"active_power_w"`
	TotalPowerImportkWh float64 `json:"total_power_import_kwh"`
	TotalPowerExportkWh float64 `json:"total_power_export_kwh"`
	ActiveCurrentA      float64 `json:"active_current_a"`
	ActiveCurrentL1A    float64 `json:"active_current_l1_a"`
	ActiveCurrentL2A    float64 `json:"active_current_l2_a"`
	ActiveCurrentL3A    float64 `json:"active_current_l3_a"`
	ActiveVoltageV      float64 `json:"active_voltage_v"`
	ActiveVoltageL1V    float64 `json:"active_voltage_l1_v"`
	ActiveVoltageL2V    float64 `json:"active_voltage_l2_v"`
	ActiveVoltageL3V    float64 `json:"active_voltage_l3_v"`
}

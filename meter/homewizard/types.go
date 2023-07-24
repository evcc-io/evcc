package homewizard

// StateResponse returns the actual state of the HomeWizard Energy Socket
// https://homewizard-energy-api.readthedocs.io/endpoints.html#recent-measurement-api-v1-data
type StateResponse struct {
	PowerOn bool `json:"power_on"`
}

// DataResponse returns the most recent measurements from the HomeWizard device
// https://homewizard-energy-api.readthedocs.io/endpoints.html#state-api-v1-state
type DataResponse struct {
	ActivePowerW          float64 `json:"active_power_w"`
	TotalPowerImportT1kWh float64 `json:"total_power_import_t1_kwh"`
	TotalPowerImportT2kWh float64 `json:"total_power_import_t2_kwh"`
	TotalPowerImportT3kWh float64 `json:"total_power_import_t3_kwh"`
	TotalPowerImportT4kWh float64 `json:"total_power_import_t4_kwh"`
}

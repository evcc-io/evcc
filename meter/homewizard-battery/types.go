package battery

// DeviceInfo returns basic device information
// https://api-documentation.homewizard.com/docs/v2/device_information
type DeviceInfo struct {
	ProductName     string `json:"product_name"`
	ProductType     string `json:"product_type"`
	Serial          string `json:"serial"`
	FirmwareVersion string `json:"firmware_version"`
	APIVersion      string `json:"api_version"`
}

// Measurement returns the most recent measurements from HomeWizard devices
// Supports both HWE-P1 (P1 Meter) and HWE-BAT (Plug-In Battery)
// https://api-documentation.homewizard.com/docs/v2/measurement
type Measurement struct {
	// Common fields
	PowerW          float64 `json:"power_w"`
	EnergyImportKWh float64 `json:"energy_import_kwh"`
	EnergyExportKWh float64 `json:"energy_export_kwh"`
	VoltageV        float64 `json:"voltage_v"`
	CurrentA        float64 `json:"current_a"`
	FrequencyHz     float64 `json:"frequency_hz"`

	// Battery-specific fields (HWE-BAT only)
	StateOfChargePct float64 `json:"state_of_charge_pct"`
	Cycles           int     `json:"cycles"`

	// P1 meter per-phase fields (HWE-P1 only)
	PowerL1W float64 `json:"power_l1_w"`
	PowerL2W float64 `json:"power_l2_w"`
	PowerL3W float64 `json:"power_l3_w"`
}

// Status returns the battery system status from P1 meter
// https://api-documentation.homewizard.com/docs/v2/batteries
type Status struct {
	Mode            string  `json:"mode"`              // "zero", "to_full", or "standby"
	PowerW          float64 `json:"power_w"`           // Combined power from all batteries
	TargetPowerW    float64 `json:"target_power_w"`    // Target power level
	MaxConsumptionW float64 `json:"max_consumption_w"` // Max charging capacity
	MaxProductionW  float64 `json:"max_production_w"`  // Max discharging capacity
}

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

// Measurement returns battery measurements from HWE-BAT
// https://api-documentation.homewizard.com/docs/v2/measurement
type Measurement struct {
	StateOfChargePct float64 `json:"state_of_charge_pct"`
}

// Status returns the battery system status from P1 meter
// https://api-documentation.homewizard.com/docs/v2/batteries
type Status struct {
	Mode   string  `json:"mode"`    // "zero", "to_full", or "standby"
	PowerW float64 `json:"power_w"` // Combined power from all batteries
}

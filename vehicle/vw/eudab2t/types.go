package eudab2t

// VINList is the request body for adding and removing vins from a subscription.
type VINList struct {
	Add    []string `json:"add,omitempty"`
	Remove []string `json:"remove,omitempty"`
}

// ConsentInfo is a single vehicle's consent status as returned by the status endpoint.
type ConsentInfo struct {
	Vehicle string `json:"vehicle"`
	Status  string `json:"status"`
}

// EU Data Act data field names as delivered by the pull api. These mirror the
// dataset field names used by the consumer portal client (vehicle/vw/eudataact).
const (
	FieldBatteryStateReportSoc = "battery_state_report.soc"
	FieldSoc                   = "state_of_charge"
	FieldHvSoc                 = "hv_soc"
	FieldHvBatteryLevel        = "battery_level_HV.value"
	FieldRangeCombined         = "cruising_range_combined"
	FieldRangePrimary          = "cruising_range_primary_engine"
	FieldRangeSecondary        = "cruising_range_secondary_engine"
	FieldOdometer              = "mileage"
	FieldOdometerValue         = "mileage.value"
	FieldChargingState         = "charging_state"
	FieldCurrentChargeState    = "charging_state_report.current_charge_state"
	FieldPlugState             = "plug_state"
	FieldTargetSoc             = "settings.target_soc"
	FieldRemainingTime         = "remaining_charging_time"
)

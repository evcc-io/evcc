package gen2

const (
	InfoPath    = "info"
	ControlPath = "control"
	ValuesPath  = "values"
)

type Info struct {
	Connector *Connector `json:"connector"`
	Grid      *Grid      `json:"grid"`
}

type Connector struct {
	PhaseCount uint8 `json:"phase_count"`
	// Note: there is a bug in the notation in the API at the moment 'voltage:' with colon is expected here!
	MaxCurrent float64 `json:"max_current:"`
	Type       string  `json:"type"`
	Serial     string  `json:"serial"`
}

type Grid struct {
	Voltage uint8 `json:"voltage"`
	// Note: there is a bug in the notation in the API at the moment 'voltage:' with colon is expected here!
	Frequency int `json:"frequency:"`
	// 0 - "UNKNOWN",
	// 1 - "L1",
	// 2 - "L2",
	// 3 - "L1, L2",
	// 4 - "L3",
	// 5 - "L1, L3",
	// 6 - "L2, L3",
	// 7 - "L1, L2, L3"
	Phases string `json:"phases"`
}

type Control struct {
	CurrentSet  float64 `json:"current_set,omitempty"`
	ChargePause uint8   `json:"charge_pause,omitempty"`
	EnergyLimit uint32  `json:"energy_limit,omitempty"`
	PhaseCount  uint8   `json:"phase_count,omitempty"`
	Response    string  `json:"omitempty"` // api message if not ok
}

// Values is Settings.Values
type Values struct {
	Energy       Energy       `json:"energy"`
	Powerflow    Powerflow    `json:"powerflow"`
	General      General      `json:"general"`
	Temperatures Temperatures `json:"temperatures"`
}

type Energy struct {
	// Total charged energy overall in Watt-hours
	TotalChargedEnergy uint64 `json:"total_charged_energy"`
	ChargedEnergy      uint32 `json:"charged_energy"`
}

type Powerflow struct {
	ChargingVoltage float64 `json:"charging_voltage"`
	ChargingCurrent float64 `json:"charging_current"`
	GridFrequency   float64 `json:"grid_frequency"`
	PeakPower       float64 `json:"peak_power"`
	// Active power of all phases combined in Watt
	TotalActivePower   float64 `json:"total_active_power"`
	TotalReactivePower float64 `json:"total_reactive_power"`
	TotalApparentPower float64 `json:"total_apparent_power"`
	TotalPowerFactor   float64 `json:"total_power_factor"`
	L1                 Phase   `json:"l1"`
	L2                 Phase   `json:"l2"`
	L3                 Phase   `json:"l3"`
	N                  N       `json:"n"`
}

type N struct {
	Current float64 `json:"current"`
}

type Phase struct {
	// Note: there is a bug in the notation in the API at the moment 'voltage:' with colon is expected here!
	Voltage       float64 `json:"voltage:"`
	Current       float64 `json:"current"`
	ActivePower   float64 `json:"active_power"`
	ReactivePower float64 `json:"reactive_power"`
	ApparentPower float64 `json:"apparent_power"`
	PowerFactor   float64 `json:"power_factor"`
}

type General struct {
	ChargingRate        float64 `json:"charging_rate"`
	VehicleConnectTime  uint    `json:"vehicle_connect_time"`
	VehicleChargingTime uint    `json:"vehicle_charging_time"`
	// 0 - "UNKNOWN",
	// 1 - "STANDBY",
	// 2 - "CONNECTED",
	// 3 - "CHARGING",
	// 6 - "ERROR",
	// 7 - "WAKEUP"
	Status          string `json:"status"`
	ChargePermitted int    `json:"charge_permitted"`
	RelayState      string `json:"relay_state"`
	ChargeCount     uint16 `json:"charge_count"`
	// 0 - "NO_FAULT",
	// 1 - "AC_30MA_FAULT",
	// 2 - "AC_60MA_FAULT",
	// 3 - "AC_150MA_FAULT",
	// 4 - "DC_POSITIVE_6MA_FAULT",
	// 5 - "DC_NEGATIVE_6MA_FAULT",
	// x - "UNKNOWN"
	RcdTrigger string `json:"rcd_trigger"`
	// 0 - "NO_WARNING",
	// 1 - "NO_PE",
	// 2 - "BLACKOUT_PROTECTION",
	// 3 - "ENERGY_LIMIT_REACHED",
	// 4 - "EV_DOES_NOT_COMPLY_STANDARD",
	// 5 - "UNSUPPORTED_CHARGING_MODE",
	// 6 - "NO_ATTACHMENT_DETECTED",
	// 7 - "NO_COMM_WITH_TYPE2_ATTACHMENT",
	// 16 - "INCREASED_TEMPERATURE",
	// 17 - "INCREASED_HOUSING_TEMPERATURE",
	// 18 - "INCREASED_ATTACHMENT_TEMPERATURE",
	// 19 - "INCREASED_DOMESTIC_PLUG_TEMPERATURE",
	// x - "UNKNOWN"
	WarningCode string `json:"warning_code"`
	// 0 - "NO_ERROR",
	// 1 - "GENERAL_ERROR",
	// 2 - "32A_ATTACHMENT_ON_16A_UNIT",
	// 3 - "VOLTAGE_DROP_DETECTED",
	// 4 - "UNPLUG_DETECTION_TRIGGERED",
	// 5 - "TYPE2_NOT_AUTHORIZED",
	// 16 - "RESIDUAL_CURRENT_DETECTED",
	// 32 - "CP_SIGNAL_VOLTAGE_ERROR",
	// 33 - "CP_SIGNAL_IMPERMISSIBLE",
	// 34 - "EV_DIODE_FAULT",
	// 48 - "PE_SELF_TEST_FAILED",
	// 49 - "RCD_SELF_TEST_FAILED",
	// 50 - "RELAY_SELF_TEST_FAILED",
	// 51 - "PE_AND_RCD_SELF_TEST_FAILED",
	// 52 - "PE_AND_RELAY_SELF_TEST_FAILED",
	// 53 - "RCD_AND_RELAY_SELF_TEST_FAILED",
	// 54 - "PE_AND_RCD_AND_RELAY_SELF_TEST_FAILED",
	// 64 - "SUPPLY_VOLTAGE_ERROR",
	// 65 - "PHASE_SHIFT_ERROR",
	// 66 - "OVERVOLTAGE_DETECTED",
	// 67 - "UNDERVOLTAGE_DETECTED",
	// 68 - "OVERVOLTAGE_WITHOUT_PE_DETECTED",
	// 69 - "UNDERVOLTAGE_WITHOUT_PE_DETECTED",
	// 70 - "UNDERFREQUENCY_DETECTED",
	// 71 - "OVERFREQUENCY_DETECTED",
	// 72 - "UNKNOWN_FREQUENCY_TYPE",
	// 73 - "UNKNOWN_GRID_TYPE",
	// 80 - "GENERAL_OVERTEMPERATURE",
	// 81 - "HOUSING_OVERTEMPERATURE",
	// 82 - "ATTACHMENT_OVERTEMPERATURE",
	// 83 - "DOMESTIC_PLUG_OVERTEMPERATURE",
	// x - "UNKNOWN"
	ErrorCode string `json:"error_code"`
}

type Temperatures struct {
	Housing       float64 `json:"housing"`
	ConnectorL1   float64 `json:"connector_l1"`
	ConnectorL2   float64 `json:"connector_l2"`
	ConnectorL3   float64 `json:"connector_l3"`
	DomesticPlug1 float64 `json:"domestic_plug_1"`
	DomesticPlug2 float64 `json:"domestic_plug_2"`
}

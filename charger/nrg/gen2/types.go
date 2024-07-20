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
	PhaseCount uint8   `json:"phase_count"`
	MaxCurrent float64 `json:"max_current"`
	Type       string  `json:"type"`
	Serial     string  `json:"serial"`
}

type Grid struct {
	Voltage   uint8 `json:"voltage"`
	Frequency int   `json:"frequency"`
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
	Energy       *Energy       `json:"energy"`
	Powerflow    *Powerflow    `json:"powerflow"`
	General      *General      `json:"general"`
	Temperatures *Temperatures `json:"temperatures"`
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
	L1                 *Phase  `json:"l1"`
	L2                 *Phase  `json:"l2"`
	L3                 *Phase  `json:"l3"`
	N                  *N      `json:"n"`
}

type N struct {
	Current float64 `json:"current"`
}

type Phase struct {
	Voltage       float64 `json:"voltage"`
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
	RcdTrigger      string `json:"rcd_trigger"`
	WarningCode     string `json:"warning_code"`
	ErrorCode       string `json:"error_code"`
}

type Temperatures struct {
	Housing       float64 `json:"housing"`
	ConnectorL1   float64 `json:"connector_l1"`
	ConnectorL2   float64 `json:"connector_l2"`
	ConnectorL3   float64 `json:"connector_l3"`
	DomesticPlug1 float64 `json:"domestic_plug_1"`
	DomesticPlug2 float64 `json:"domestic_plug_2"`
}

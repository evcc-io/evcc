package pro

// https://openwb.de/main/?page_id=771

type Status struct {
	Date           string
	Timestamp      int64
	Currents       []float64
	Powers         []float64
	PowerAll       float64 `json:"power_all"`
	Imported       float64
	Exported       float64
	PlugState      bool    `json:"plug_state"`
	ChargeState    bool    `json:"charge_state"`
	PhasesActual   int     `json:"phases_actual"`
	PhasesTarget   int     `json:"phases_target"`
	PhasesInUse    int     `json:"phases_in_use"`
	OfferedCurrent float64 `json:"offered_current"`
	EvseSignaling  string  `json:"evse_signaling"`
	VehicleID      string  `json:"vehicle_id"`
	Soc            int     `json:"soc_value"`
	SocTimestamp   int64   `json:"soc_timestamp"`
	Serial         string
}

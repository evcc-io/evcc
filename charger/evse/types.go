package evse

const Success = "S0_"

// ParameterResponse is the getParameters response
type ParameterResponse struct {
	Type string      `json:"type"`
	List []ListEntry `json:"list"`
}

// ListEntry is ParameterResponse.List
type ListEntry struct {
	VehicleState    int64   `json:"vehicleState"`
	EvseState       bool    `json:"evseState"`
	MaxCurrent      int64   `json:"maxCurrent"`
	ActualCurrent   int64   `json:"actualCurrent"`
	ActualCurrentMA *int64  `json:"actualCurrentMA"` // 1/100 A
	ActualPower     float64 `json:"actualPower"`
	Duration        int64   `json:"duration"`
	AlwaysActive    bool    `json:"alwaysActive"`
	UseMeter        bool    `json:"useMeter"`
	LastActionUser  string  `json:"lastActionUser"`
	LastActionUID   string  `json:"lastActionUID"`
	Energy          float64 `json:"energy"`
	Mileage         float64 `json:"mileage"`
	MeterReading    float64 `json:"meterReading"`
	CurrentP1       float64 `json:"currentP1"`
	CurrentP2       float64 `json:"currentP2"`
	CurrentP3       float64 `json:"currentP3"`
	VoltageP1       float64 `json:"voltageP1"`
	VoltageP2       float64 `json:"voltageP2"`
	VoltageP3       float64 `json:"voltageP3"`
	RFIDUID         *string `json:"RFIDUID"`
}

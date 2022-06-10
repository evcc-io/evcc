package connectiq

type ChargeStatus struct {
	Amps   int64  `json:"amps"`
	Pp     int64  `json:"pp"`
	Status string `json:"status"`
	Std    int64  `json:"std"`
}

type ChargeMaxAmps struct {
	Max int64 `json:"max"`
}

type MeterStatus struct {
	App  []float64 `json:"app"`
	Curr []float64 `json:"curr"`
	Fac  []float64 `json:"fac"`
	Pow  []float64 `json:"pow"`
	Volt []float64 `json:"volt"`
}

type MeterRead struct {
	Energy float64 `json:"energy"`
}

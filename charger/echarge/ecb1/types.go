package ecb1

type Meter struct {
	ID   int
	Name string
	Data map[string]float64
}

type ChargeControl struct {
	ID            int
	Name          string
	State         string
	Mode          string
	ManualModeAmp float64
}

type Rfid struct {
	ID   int
	Name string
	Data struct {
		Tag string
	}
}

type All struct {
	ProtocolVersion string `json:"protocol-version"`
	Network         struct{}
	System          struct{}
	Meters          []Meter
	ChargeControls  []ChargeControl
	Rfid            []Rfid
}

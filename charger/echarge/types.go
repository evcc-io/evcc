package echarge

const (
	ModeEco    = "eco"
	ModeManual = "manual"
)

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
}

type All struct {
	Network        struct{}
	System         struct{}
	Meters         Meters
	ChargeControls ChargeControls
}

type Meters []Meter

func (s Meters) ByName(name string) Meter {
	for _, e := range s {
		if e.Name == name {
			return e
		}
	}
	return Meter{}
}

type ChargeControls []ChargeControl

func (s ChargeControls) ByName(name string) ChargeControl {
	for _, e := range s {
		if e.Name == name {
			return e
		}
	}
	return ChargeControl{}
}

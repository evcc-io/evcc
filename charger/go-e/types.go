package goe

// CloudResponse is the cloud API response
type CloudResponse struct {
	Success *bool // only valid for cloud payload commands
	Age     int
	Error   string // only valid for cloud payload commands
	Data    StatusResponse
}

// StatusResponse is the API response if status not OK
type StatusResponse struct {
	Fwv string    `json:"fwv"`        // firmware version
	Car int       `json:"car,string"` // car status
	Alw int       `json:"alw,string"` // allow charging
	Amp int       `json:"amp,string"` // current [A]
	Err int       `json:"err,string"` // error
	Eto int       `json:"eto,string"` // energy total [0.1kWh]
	Stp int       `json:"stp,string"` // stop state
	Tmp int       `json:"tmp,string"` // temperature [°C]
	Dws int       `json:"dws,string"` // energy [Ws]
	Nrg []float64 `json:"nrg"`        // voltage, current, power
	Uby int       `json:"uby,string"` // unlocked_by
	Rna string    `json:"rna"`        // RFID 1
	Rnm string    `json:"rnm"`        // RFID 2
	Rne string    `json:"rne"`        // RFID 3
	Rn4 string    `json:"rn4"`        // RFID 4
	Rn5 string    `json:"rn5"`        // RFID 5
	Rn6 string    `json:"rn6"`        // RFID 6
	Rn7 string    `json:"rn7"`        // RFID 7
	Rn8 string    `json:"rn8"`        // RFID 8
	Rn9 string    `json:"rn9"`        // RFID 9
	Rn1 string    `json:"rn1"`        // RFID 10
}

func (g *StatusResponse) Status() int {
	return g.Car
}

func (g *StatusResponse) Enabled() bool {
	return g.Alw == 1
}

func (g *StatusResponse) CurrentPower() float64 {
	if len(g.Nrg) == 16 {
		return g.Nrg[11] * 10
	}
	return 0
}

func (g *StatusResponse) ChargedEnergy() float64 {
	return float64(g.Dws) / 3.6e5 // Deka-Watt-Seconds to kWh (100.000 == 0,277kWh)
}

func (g *StatusResponse) TotalEnergy() float64 {
	return float64(g.Eto) / 1e1 // 0.1kWh to kWh (130 == 13kWh)
}

func (g *StatusResponse) Currents() (float64, float64, float64) {
	if len(g.Nrg) == 16 {
		return g.Nrg[4] / 10, g.Nrg[5] / 10, g.Nrg[6] / 10
	}
	return 0, 0, 0
}

func (g *StatusResponse) Voltages() (float64, float64, float64) {
	if len(g.Nrg) == 16 {
		return g.Nrg[0], g.Nrg[1], g.Nrg[2]
	}
	return 0, 0, 0
}

func (g *StatusResponse) Identify() string {
	switch g.Uby {
	case 1:
		return g.Rna
	case 2:
		return g.Rnm
	case 3:
		return g.Rne
	case 4:
		return g.Rn4
	case 5:
		return g.Rn5
	case 6:
		return g.Rn6
	case 7:
		return g.Rn7
	case 8:
		return g.Rn8
	case 9:
		return g.Rn9
	case 10:
		return g.Rn1
	default:
		return ""
	}
}

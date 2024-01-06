package goe

// StatusResponse2 is the v2 API response
type StatusResponse2 struct {
	Fwv   string    // firmware version
	Car   int       // car status
	Alw   bool      // allow charging
	Amp   int       // current [A]
	Err   int       // error
	Eto   uint64    // energy total Wh
	Psm   int       // phase switching
	Stp   int       // stop state
	Tmp   int       // temperature [°C]
	Trx   int       // transaction
	Nrg   []float64 // voltage, current, power
	Wh    float64   // energy [Wh]
	Cards []Card    // RFID cards
}

// Card is the v2 RFID card
type Card struct {
	Name   string
	Energy float64
	CardID bool
}

func (g *StatusResponse2) Status() int {
	return g.Car
}

func (g *StatusResponse2) Enabled() bool {
	return g.Alw
}

func (g *StatusResponse2) CurrentPower() float64 {
	if len(g.Nrg) == 16 {
		return g.Nrg[11]
	}

	return 0
}

func (g *StatusResponse2) ChargedEnergy() float64 {
	return g.Wh / 1e3
}

func (g *StatusResponse2) TotalEnergy() float64 {
	return float64(g.Eto) / 1e3
}

func (g *StatusResponse2) Currents() (float64, float64, float64) {
	if len(g.Nrg) == 16 {
		return g.Nrg[4], g.Nrg[5], g.Nrg[6]
	}

	return 0, 0, 0
}

func (g *StatusResponse2) Voltages() (float64, float64, float64) {
	if len(g.Nrg) == 16 {
		return g.Nrg[0], g.Nrg[1], g.Nrg[2]
	}

	return 0, 0, 0
}

func (g *StatusResponse2) Identify() string {
	if g.Trx > 0 {
		return g.Cards[g.Trx-1].Name
	}

	return ""
}

package shelly

type ShellyMeter struct {
	Conn *Connection
}

func NewShellyMeter(conn *Connection) *ShellyMeter {
	shellyMeter := &ShellyMeter{
		Conn: conn,
	}

	return shellyMeter
}

// CurrentPower implements the api.Meter interface
func (shellyMeter *ShellyMeter) CurrentPower() (float64, error) {
	var power float64

	var res Gen2EmStatusResponse
	if err := shellyMeter.Conn.execGen2Cmd("EM.GetStatus", false, &res); err != nil {
		return 0, err
	}

	power = res.TotalPower

	return power, nil
}

// TotalEnergy implements the api.Meter interface
func (shellyMeter *ShellyMeter) TotalEnergy() (float64, error) {
	var energy float64
	var res Gen2EmDataStatusResponce
	if err := shellyMeter.Conn.execGen2Cmd("EMData.GetStatus", false, &res); err != nil {
		return 0, err
	}

	energy = res.TotalEnergy

	return energy, nil
}

// Currents implements the api.Meter interface (PhaseCurrents provides per-phase current A)
func (shellyMeter *ShellyMeter) Currents() (float64, float64, float64, error) {
	var res Gen2EmStatusResponse
	if err := shellyMeter.Conn.execGen2Cmd("EM.GetStatus", false, &res); err != nil {
		return 0, 0, 0, err
	}

	return res.CurrentA, res.CurrentB, res.CurrentC, nil
}

// Voltages implements the api.Meter interface (PhaseVoltages provides per-phase voltage V)
func (shellyMeter *ShellyMeter) Voltages() (float64, float64, float64, error) {
	var res Gen2EmStatusResponse
	if err := shellyMeter.Conn.execGen2Cmd("EM.GetStatus", false, &res); err != nil {
		return 0, 0, 0, err
	}

	return res.VoltageA, res.VoltageB, res.VoltageC, nil
}

// Powers implements the api.Meter interface (PhasePowers provides signed per-phase power W)
func (shellyMeter *ShellyMeter) Powers() (float64, float64, float64, error) {
	var res Gen2EmStatusResponse
	if err := shellyMeter.Conn.execGen2Cmd("EM.GetStatus", false, &res); err != nil {
		return 0, 0, 0, err
	}

	return res.PowerA, res.PowerB, res.PowerC, nil
}

package shelly

import "github.com/evcc-io/evcc/api"

type EnergyMeter struct {
	conn *Connection
}

func NewEnergyMeter(conn *Connection) *EnergyMeter {
	res := &EnergyMeter{
		conn: conn,
	}

	return res
}

// CurrentPower implements the api.Meter interface
func (sh *EnergyMeter) CurrentPower() (float64, error) {
	res, err := sh.conn.gen2EMStatus.Get()
	if err != nil {
		return 0, err
	}
	return res.TotalActPower, nil
}

// TotalEnergy implements the api.Meter interface
func (sh *EnergyMeter) TotalEnergy() (float64, error) {
	var res Gen2EmDataStatusResponse
	if err := sh.conn.gen2ExecCmd("EMData.GetStatus", false, &res); err != nil {
		return 0, err
	}

	return res.TotalEnergy / 1000, nil
}

var _ api.PhaseCurrents = (*EnergyMeter)(nil)

// Currents implements the api.PhaseCurrents interface
func (sh *EnergyMeter) Currents() (float64, float64, float64, error) {
	res, err := sh.conn.gen2EMStatus.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	return res.ACurrent, res.BCurrent, res.CCurrent, nil
}

var _ api.PhaseVoltages = (*EnergyMeter)(nil)

// Voltages implements the api.PhaseVoltages interface
func (sh *EnergyMeter) Voltages() (float64, float64, float64, error) {
	res, err := sh.conn.gen2EMStatus.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	return res.AVoltage, res.BVoltage, res.CVoltage, nil
}

var _ api.PhasePowers = (*EnergyMeter)(nil)

// Powers implements the api.PhasePowers interface
func (sh *EnergyMeter) Powers() (float64, float64, float64, error) {
	res, err := sh.conn.gen2EMStatus.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	return res.AActPower, res.BActPower, res.CActPower, nil
}

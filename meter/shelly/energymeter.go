package shelly

import "github.com/evcc-io/evcc/api"

type EnergyMeter struct {
	*Connection
}

func NewEnergyMeter(conn *Connection) *EnergyMeter {
	res := &EnergyMeter{
		Connection: conn,
	}

	return res
}

// CurrentPower implements the api.Meter interface
func (sh *EnergyMeter) CurrentPower() (float64, error) {
	var res Gen2EmStatusResponse
	if err := sh.Connection.execGen2Cmd("EM.GetStatus", false, &res); err != nil {
		return 0, err
	}

	return res.TotalPower, nil
}

// TotalEnergy implements the api.Meter interface
func (sh *EnergyMeter) TotalEnergy() (float64, error) {
	var res Gen2EmDataStatusResponse
	if err := sh.Connection.execGen2Cmd("EMData.GetStatus", false, &res); err != nil {
		return 0, err
	}

	return res.TotalEnergy, nil
}

var _ api.PhaseCurrents = (*EnergyMeter)(nil)

// Currents implements the api.PhaseCurrents interface
func (sh *EnergyMeter) Currents() (float64, float64, float64, error) {
	var res Gen2EmStatusResponse
	if err := sh.Connection.execGen2Cmd("EM.GetStatus", false, &res); err != nil {
		return 0, 0, 0, err
	}

	return res.CurrentA, res.CurrentB, res.CurrentC, nil
}

var _ api.PhaseVoltages = (*EnergyMeter)(nil)

// Voltages implements the api.PhaseVoltages interface
func (sh *EnergyMeter) Voltages() (float64, float64, float64, error) {
	var res Gen2EmStatusResponse
	if err := sh.Connection.execGen2Cmd("EM.GetStatus", false, &res); err != nil {
		return 0, 0, 0, err
	}

	return res.VoltageA, res.VoltageB, res.VoltageC, nil
}

var _ api.PhasePowers = (*EnergyMeter)(nil)

// Powers implements the api.PhasePowers interface
func (sh *EnergyMeter) Powers() (float64, float64, float64, error) {
	var res Gen2EmStatusResponse
	if err := sh.Connection.execGen2Cmd("EM.GetStatus", false, &res); err != nil {
		return 0, 0, 0, err
	}

	return res.PowerA, res.PowerB, res.PowerC, nil
}

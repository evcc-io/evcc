package wrapper

import "github.com/evcc-io/evcc/api"

type StatusWithCurrents interface {
	api.ChargeState
	api.PhaseCurrents
}

// ChargePhaseMeter is a replacement for a physical charge meter.
// It uses the charger's actual or max current to calculate power consumption.
type ChargePhaseMeter struct {
	Voltage float64
	Charger StatusWithCurrents
}

// CurrentPower implements the api.Meter interface
func (m *ChargePhaseMeter) CurrentPower() (float64, error) {
	status, err := m.Charger.Status()
	if status != api.StatusC || err != nil {
		return 0, err
	}

	i1, i2, i3, err := m.Charger.Currents()
	return m.Voltage * (i1 + i2 + i3), err
}

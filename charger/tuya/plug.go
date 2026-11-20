package tuya

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// standard data points of Tuya sockets with energy monitoring
const (
	plugDpSwitch  = "1"
	plugDpCurrent = "18" // mA
	plugDpPower   = "19" // 0.1 W
	plugDpVoltage = "20" // 0.1 V
)

// Plug is a Tuya smart plug with energy monitoring
type Plug struct {
	conn *Connection
}

// NewPlug creates a Tuya smart plug
func NewPlug(ctx context.Context, cc Config) (*Plug, error) {
	conn, _, err := Dial(ctx, util.NewLogger("tuya").Redact(cc.LocalKey), cc)
	if err != nil {
		return nil, err
	}

	return &Plug{conn: conn}, nil
}

func plugValue(dps map[string]any, dp string, scale float64) (float64, error) {
	v, ok := dps[dp].(float64)
	if !ok {
		return 0, api.ErrNotAvailable
	}

	return v / scale, nil
}

func (p *Plug) value(dp string, scale float64) (float64, error) {
	dps, err := p.conn.Dps()
	if err != nil {
		return 0, err
	}

	return plugValue(dps, dp, scale)
}

// Enabled implements the api.Charger interface
func (p *Plug) Enabled() (bool, error) {
	dps, err := p.conn.Dps()
	if err != nil {
		return false, err
	}

	on, ok := dps[plugDpSwitch].(bool)
	if !ok {
		return false, api.ErrNotAvailable
	}

	return on, nil
}

// Enable implements the api.Charger interface
func (p *Plug) Enable(enable bool) error {
	return p.conn.Set(map[string]any{plugDpSwitch: enable})
}

var _ api.Meter = (*Plug)(nil)

// CurrentPower implements the api.Meter interface
func (p *Plug) CurrentPower() (float64, error) {
	return p.value(plugDpPower, 10)
}

var _ api.PhaseCurrents = (*Plug)(nil)

// Currents implements the api.PhaseCurrents interface
func (p *Plug) Currents() (float64, float64, float64, error) {
	res, err := p.value(plugDpCurrent, 1e3)
	return res, 0, 0, err
}

var _ api.PhaseVoltages = (*Plug)(nil)

// Voltages implements the api.PhaseVoltages interface
func (p *Plug) Voltages() (float64, float64, float64, error) {
	res, err := p.value(plugDpVoltage, 10)
	return res, 0, 0, err
}

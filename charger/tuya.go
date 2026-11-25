package charger

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/tuya"
	"github.com/evcc-io/evcc/util"
)

// Tuya smart plug charger implementation
type Tuya struct {
	plug *tuya.Plug
	*switchSocket
}

func init() {
	registry.AddCtx("tuya", NewTuyaFromConfig)
}

// NewTuyaFromConfig creates a Tuya smart plug charger from generic config
func NewTuyaFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	var cc struct {
		embed        `mapstructure:",squash"`
		tuya.Config  `mapstructure:",squash"`
		StandbyPower float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	plug, err := tuya.NewPlug(ctx, cc.Config)
	if err != nil {
		return nil, err
	}

	c := &Tuya{plug: plug}
	c.switchSocket = NewSwitchSocket(&cc.embed, plug.Enabled, plug.CurrentPower, cc.StandbyPower)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *Tuya) Enabled() (bool, error) {
	return c.plug.Enabled()
}

// Enable implements the api.Charger interface
func (c *Tuya) Enable(enable bool) error {
	return c.plug.Enable(enable)
}

var _ api.PhaseCurrents = (*Tuya)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *Tuya) Currents() (float64, float64, float64, error) {
	return c.plug.Currents()
}

var _ api.PhaseVoltages = (*Tuya)(nil)

// Voltages implements the api.PhaseVoltages interface
func (c *Tuya) Voltages() (float64, float64, float64, error) {
	return c.plug.Voltages()
}

package charger

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx("switchsocket", NewSwitchSocketFromConfig)
}

type SwitchSocket struct {
	enable  func(bool) error
	enabled func() (bool, error)
	*switchSocket
}

func NewSwitchSocketFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		embed        `mapstructure:",squash"`
		Enabled      provider.Config
		Enable       provider.Config
		Power        provider.Config
		StandbyPower float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	enabled, err := provider.NewBoolGetterFromConfig(ctx, cc.Enabled)
	if err != nil {
		return nil, err
	}

	enable, err := provider.NewBoolSetterFromConfig(ctx, "enable", cc.Enable)
	if err != nil {
		return nil, err
	}

	power, err := provider.NewFloatGetterFromConfig(ctx, cc.Power)
	if err != nil {
		return nil, err
	}

	c := &SwitchSocket{
		enabled:      enabled,
		enable:       enable,
		switchSocket: NewSwitchSocket(&cc.embed, enabled, power, cc.StandbyPower),
	}

	return c, nil
}

func (c *SwitchSocket) Enabled() (bool, error) {
	return c.enabled()
}

func (c *SwitchSocket) Enable(enable bool) error {
	return c.enable(enable)
}

// switchSocket implements the api.Charger Status and CurrentPower methods
// using basic generic switch socket functions
type switchSocket struct {
	*embed
	enabled      func() (bool, error)
	currentPower func() (float64, error)
	standbypower float64
	lp           loadpoint.API
}

func NewSwitchSocket(
	embed *embed,
	enabled func() (bool, error),
	currentPower func() (float64, error),
	standbypower float64,
) *switchSocket {
	return &switchSocket{
		embed:        embed,
		enabled:      enabled,
		currentPower: currentPower,
		standbypower: standbypower,
	}
}

// Status calculates a generic switches status
func (c *switchSocket) Status() (api.ChargeStatus, error) {
	if c.lp != nil && c.lp.GetMode() == api.ModeOff {
		return api.StatusA, nil
	}

	res := api.StatusB

	// static mode
	if c.standbypower < 0 {
		on, err := c.enabled()
		if on {
			res = api.StatusC
		}

		return res, err
	}

	// standby power mode
	power, err := c.currentPower()
	if power > c.standbypower {
		res = api.StatusC
	}

	return res, err
}

// MaxCurrent implements the api.Charger interface
func (c *switchSocket) MaxCurrent(current int64) error {
	return nil
}

var _ api.ChargerEx = (*switchSocket)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *switchSocket) MaxCurrentMillis(current float64) error {
	return nil
}

var _ api.Meter = (*switchSocket)(nil)

// CurrentPower calculates a generic switches power
func (c *switchSocket) CurrentPower() (float64, error) {
	var power float64

	// set fix static power in static mode
	if c.standbypower < 0 {
		on, err := c.enabled()
		if on {
			power = -c.standbypower
		}
		return power, err
	}

	// ignore power in standby mode
	power, err := c.currentPower()
	if power <= c.standbypower {
		power = 0
	}

	return power, err
}

var _ loadpoint.Controller = (*switchSocket)(nil)

// LoadpointControl implements loadpoint.Controller
func (c *switchSocket) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}

var _ api.PhaseDescriber = (*switchSocket)(nil)

// Phases implements the api.PhasesDescriber interface
func (v *switchSocket) Phases() int {
	return 1
}

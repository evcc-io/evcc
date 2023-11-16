package charger

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

type ChargeableBattery interface {
	api.Meter
	api.Battery
	api.BatteryController
}

// Battery implements an api.Charger for controllable batteries
type Battery struct {
	*embed
	ChargeableBattery
	mode api.BatteryMode
}

func init() {
	registry.Add("battery", NewBatteryFromConfig)
}

func NewBatteryFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		embed   `mapstructure:",squash"`
		Battery string
	}{
		embed: embed{
			Icon_:     "battery",
			Features_: []api.Feature{api.IntegratedDevice},
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	dev, err := config.Meters().ByName(cc.Battery)
	if err != nil {
		return nil, err
	}

	battery, ok := dev.Instance().(ChargeableBattery)
	if !ok {
		return nil, fmt.Errorf("'%s' is not a chargeable battery", cc.Battery)
	}

	c := &Battery{
		ChargeableBattery: battery,
	}

	return c, nil
}

// Status calculates the battery charging status
func (c *Battery) Status() (api.ChargeStatus, error) {
	res := api.StatusB

	p, err := c.ChargeableBattery.CurrentPower()
	if p > 0 && err == nil {
		res = api.StatusC
	}

	return res, err
}

// Enabled implements the api.Charger interface
func (c *Battery) Enabled() (bool, error) {
	return c.mode == api.BatteryCharge, nil
}

// Enable implements the api.Charger interface
func (c *Battery) Enable(enable bool) error {
	mode := api.BatteryNormal
	if enable {
		mode = api.BatteryCharge
	}

	// TODO handle locking of battery
	err := c.ChargeableBattery.SetBatteryMode(mode)
	if err == nil {
		c.mode = mode
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *Battery) MaxCurrent(current int64) error {
	return nil
}

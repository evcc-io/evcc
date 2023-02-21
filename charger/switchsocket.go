package charger

import "github.com/evcc-io/evcc/api"

// switchSocket implements the api.Charger Status and CurrentPower methods
// using basic generic switch socket functions
type switchSocket struct {
	*embed
	enabled      func() (bool, error)
	currentPower func() (float64, error)
	standbypower float64
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

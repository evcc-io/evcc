package shelly

import (
	"errors"
	"fmt"
	"strings"
)

// CurrentPower implements the api.Meter interface
func (c *Connection) Gen1CurrentPower() (float64, error) {
	var switchpower, meterpower float64
	var res Gen1StatusResponse
	uri := fmt.Sprintf("%s/status", c.uri)
	if err := c.GetJSON(uri, &res); err != nil {
		return 0, err
	}

	switch {
	case c.channel < len(res.Meters):
		switchpower = res.Meters[c.channel].Power
	case c.channel < len(res.EMeters):
		meterpower = res.EMeters[c.channel].Power
	default:
		return 0, errors.New("invalid channel, missing power meter")
	}
	return switchpower + meterpower, nil
}

// Enabled implements the api.Charger interface
func (c *Connection) Gen1Enabled() (bool, error) {
	var res Gen1SwitchResponse
	uri := fmt.Sprintf("%s/relay/%d", c.uri, c.channel)
	err := c.GetJSON(uri, &res)
	return res.Ison, err
}

// Enable implements the api.Charger interface
func (c *Connection) Gen1Enable(enable bool) error {
	var err error
	onoff := map[bool]string{true: "on", false: "off"}

	var res Gen1SwitchResponse
	uri := fmt.Sprintf("%s/relay/%d?turn=%s", c.uri, c.channel, onoff[enable])
	err = c.GetJSON(uri, &res)

	return err
}

// TotalEnergy implements the api.Meter interface
func (c *Connection) Gen1TotalEnergy() (float64, error) {
	var energy float64

	var res Gen1StatusResponse
	uri := fmt.Sprintf("%s/status", c.uri)
	if err := c.GetJSON(uri, &res); err != nil {
		return 0, err
	}

	switch {
	case c.channel < len(res.Meters):
		energy = res.Meters[c.channel].Total - res.Meters[c.channel].Total_Returned
	case c.channel < len(res.EMeters):
		energy = res.EMeters[c.channel].Total - res.EMeters[c.channel].Total_Returned
	default:
		return 0, errors.New("invalid channel, missing power meter")
	}

	energy = gen1Energy(c.model, energy)

	return energy / 1000, nil
}

// gen1Energy in kWh
func gen1Energy(model string, energy float64) float64 {
	// Gen 1 Shelly EM devices are providing Watt hours, Gen 1 Shelly PM devices are providing Watt minutes
	if !strings.Contains(model, "EM") {
		energy /= 60
	}
	return energy
}

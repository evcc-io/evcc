package fritzdect

import (
	"strconv"
)

// CurrentPower implements the api.Meter interface
func (c *Connection) CurrentPower() (float64, error) {
	// power value in 0,001 W (current switch power, refresh approximately every 2 minutes)
	resp, err := c.ExecCmd("getswitchpower")
	if err != nil {
		return 0, err
	}

	power, err := strconv.ParseFloat(resp, 64)

	return power / 1000, err // mW ==> W
}

// CurrentPower implements the api.MeterEnergy interface
func (c *Connection) TotalEnergy() (float64, error) {
	// Energy value in Wh (total switch energy, refresh approximately every 2 minutes)
	resp, err := c.ExecCmd("getswitchenergy")
	if err != nil {
		return 0, err
	}

	energy, err := strconv.ParseFloat(resp, 64)

	return energy / 1000, err // Wh ==> KWh
}

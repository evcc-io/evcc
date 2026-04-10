package fritzdect_new

import (
	"encoding/xml"
	"strconv"
	"strings"
)

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

// CurrentPower implements the api.Meter interface
func (c *Connection) CurrentPower() (float64, error) {
	// power value in 0,01 W (current switch power, refresh approximately every 2 minutes)
	resp, err := c.ExecCmd("getbasicdevicestats")
	if err != nil {
		return 0, err
	}

	var f = ParseFXml(resp, err)
	return (f * 10) / 1000, err // 1/100W ==> W
}

func ParseFXml(s string, err error) float64 {
	var v Devicestats

	err2 := xml.Unmarshal([]byte(s), &v)
	if err2 != nil {
		//
	}

	var csv = v.Power.Values[0]

	parts := strings.Split(csv, ",")
	if len(parts) == 0 {
		//
	}

	f, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		//
	}

	return float64(f)
}

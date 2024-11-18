package shelly

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

type Switch struct {
	*Connection
}

func NewSwitch(conn *Connection) *Switch {
	res := &Switch{
		Connection: conn,
	}

	return res
}

// CurrentPower implements the api.Meter interface
func (sh *Switch) CurrentPower() (float64, error) {
	var power float64

	d := sh.Connection
	switch d.gen {
	case 0, 1:
		var res Gen1StatusResponse
		uri := fmt.Sprintf("%s/status", d.uri)
		if err := d.GetJSON(uri, &res); err != nil {
			return 0, err
		}

		switch {
		case d.channel < len(res.Meters):
			power = res.Meters[d.channel].Power
		case d.channel < len(res.EMeters):
			power = res.EMeters[d.channel].Power
		default:
			return 0, errors.New("invalid channel, missing power meter")
		}

	default:
		var resem Gen2EmStatusResponse
		var res Gen2StatusResponse
		if d.app == "Pro3EM" && d.profile == "monophase" {
			if err := d.execGen2Cmd("Shelly.GetStatus", false, &resem); err != nil {
				return 0, err
			}
		} else {
			if err := d.execGen2Cmd("Shelly.GetStatus", false, &res); err != nil {
				return 0, err
			}
		}

		switch d.channel {
		case 1:
			power = res.Switch1.Apower + res.Pm1.Apower + resem.Em1.ActPower
		case 2:
			power = res.Switch2.Apower + res.Pm2.Apower + resem.Em2.ActPower
		default:
			power = res.Switch0.Apower + res.Pm0.Apower + resem.Em0.ActPower
		}
	}

	// Assure positive power response (Gen 1 EM devices can provide negative values)
	return math.Abs(power), nil
}

// Enabled implements the api.Charger interface
func (sh *Switch) Enabled() (bool, error) {
	d := sh.Connection
	switch d.gen {
	case 0, 1:
		var res Gen1SwitchResponse
		uri := fmt.Sprintf("%s/relay/%d", d.uri, d.channel)
		err := d.GetJSON(uri, &res)
		return res.Ison, err

	default:
		var res Gen2SwitchResponse
		err := d.execGen2Cmd("Switch.GetStatus", false, &res)
		return res.Output, err
	}
}

// Enable implements the api.Charger interface
func (sh *Switch) Enable(enable bool) error {
	var err error
	onoff := map[bool]string{true: "on", false: "off"}

	d := sh.Connection
	switch d.gen {
	case 0, 1:
		var res Gen1SwitchResponse
		uri := fmt.Sprintf("%s/relay/%d?turn=%s", d.uri, d.channel, onoff[enable])
		err = d.GetJSON(uri, &res)

	default:
		var res Gen2SwitchResponse
		err = d.execGen2Cmd("Switch.Set", enable, &res)
	}

	return err
}

// TotalEnergy implements the api.Meter interface
func (sh *Switch) TotalEnergy() (float64, error) {
	var energy float64

	d := sh.Connection
	switch d.gen {
	case 0, 1:
		var res Gen1StatusResponse
		uri := fmt.Sprintf("%s/status", d.uri)
		if err := d.GetJSON(uri, &res); err != nil {
			return 0, err
		}

		switch {
		case d.channel < len(res.Meters):
			energy = res.Meters[d.channel].Total
		case d.channel < len(res.EMeters):
			energy = res.EMeters[d.channel].Total
		default:
			return 0, errors.New("invalid channel, missing power meter")
		}

		energy = gen1Energy(d.devicetype, energy)

	default:
		var resem Gen2EmStatusResponse
		var res Gen2StatusResponse
		if d.app == "Pro3EM" && d.profile == "monophase" {
			if err := d.execGen2Cmd("Shelly.GetStatus", false, &resem); err != nil {
				return 0, err
			}
		} else {
			if err := d.execGen2Cmd("Shelly.GetStatus", false, &res); err != nil {
				return 0, err
			}
		}

		switch d.channel {
		case 1:
			energy = res.Switch1.Aenergy.Total + res.Pm1.Aenergy.Total + resem.Em1Data.TotalActEnergy - resem.Em1Data.TotalActRetEnergy
		case 2:
			energy = res.Switch2.Aenergy.Total + res.Pm2.Aenergy.Total + resem.Em2Data.TotalActEnergy - resem.Em2Data.TotalActRetEnergy
		default:
			energy = res.Switch0.Aenergy.Total + res.Pm0.Aenergy.Total + resem.Em0Data.TotalActEnergy - resem.Em0Data.TotalActRetEnergy
		}
	}

	return energy / 1000, nil
}

// gen1Energy in kWh
func gen1Energy(devicetype string, energy float64) float64 {
	// Gen 1 Shelly EM devices are providing Watt hours, Gen 1 Shelly PM devices are providing Watt minutes
	if !strings.Contains(devicetype, "EM") {
		energy /= 60
	}
	return energy
}

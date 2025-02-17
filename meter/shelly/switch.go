package shelly

import (
	"errors"
	"fmt"
	"strings"
)

type Switch struct {
	*Connection
	Usage  string
	Invert bool
}

func NewSwitch(conn *Connection, usage string, invert bool) *Switch {
	res := &Switch{
		Connection: conn,
		Usage:      usage,
		Invert:     invert,
	}

	return res
}

// CurrentPower implements the api.Meter interface
func (sh *Switch) CurrentPower() (float64, error) {
	var switchpower float64
	var meterpower float64

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
			switchpower = res.Meters[d.channel].Power
		case d.channel < len(res.EMeters):
			meterpower = res.EMeters[d.channel].Power
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
			switchpower = res.Switch1.Apower
			meterpower = res.Pm1.Apower + resem.Em1.ActPower
		case 2:
			switchpower = res.Switch2.Apower
			meterpower = res.Pm2.Apower + resem.Em2.ActPower
		default:
			switchpower = res.Switch0.Apower
			meterpower = res.Pm0.Apower + resem.Em0.ActPower
		}
	}

	if (sh.Usage == "pv" || sh.Usage == "battery") && !sh.Invert {
		meterpower = -meterpower
	}

	return switchpower + meterpower, nil
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
			if (sh.Usage == "pv" || sh.Usage == "battery") && !sh.Invert {
				energy = res.Meters[d.channel].Total_Returned
			} else {
				energy = res.Meters[d.channel].Total
			}
		case d.channel < len(res.EMeters):
			if (sh.Usage == "pv" || sh.Usage == "battery") && !sh.Invert {
				energy = res.EMeters[d.channel].Total_Returned
			} else {
				energy = res.EMeters[d.channel].Total
			}
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

		if (sh.Usage == "pv" || sh.Usage == "battery") && !sh.Invert {
			switch d.channel {
			case 1:
				energy = res.Switch1.Aenergy.Total + res.Pm1.Ret_Aenergy.Total + resem.Em1Data.TotalActRetEnergy
			case 2:
				energy = res.Switch2.Aenergy.Total + res.Pm2.Ret_Aenergy.Total + resem.Em2Data.TotalActRetEnergy
			default:
				energy = res.Switch0.Aenergy.Total + res.Pm0.Ret_Aenergy.Total + resem.Em0Data.TotalActRetEnergy
			}
		} else {
			switch d.channel {
			case 1:
				energy = res.Switch1.Aenergy.Total + res.Pm1.Aenergy.Total + resem.Em1Data.TotalActEnergy
			case 2:
				energy = res.Switch2.Aenergy.Total + res.Pm2.Aenergy.Total + resem.Em2Data.TotalActEnergy
			default:
				energy = res.Switch0.Aenergy.Total + res.Pm0.Aenergy.Total + resem.Em0Data.TotalActEnergy
			}
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

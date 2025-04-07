package shelly

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

type Switch struct {
	conn *Connection
}

func NewSwitch(conn *Connection) *Switch {
	res := &Switch{
		conn: conn,
	}

	return res
}

// CurrentPower implements the api.Meter interface
func (sh *Switch) CurrentPower() (float64, error) {
	var power float64

	switch sh.conn.gen {
	case 0, 1:
		var res Gen1Status
		uri := fmt.Sprintf("%s/status", sh.conn.uri)
		if err := sh.conn.GetJSON(uri, &res); err != nil {
			return 0, err
		}

		switch {
		case sh.conn.channel < len(res.Meters):
			power = res.Meters[sh.conn.channel].Power
		case sh.conn.channel < len(res.EMeters):
			power = res.EMeters[sh.conn.channel].Power
		default:
			return 0, errors.New("invalid channel, missing power meter")
		}

	default:
		var res Gen2StatusResponse
		if err := sh.conn.execGen2Cmd("Shelly.GetStatus", false, &res); err != nil {
			return 0, err
		}

		switch sh.conn.channel {
		case 1:
			power = res.Switch1.Apower + res.Pm1.Apower + res.Em1.ActPower
		case 2:
			power = res.Switch2.Apower + res.Pm2.Apower + res.Em2.ActPower
		default:
			power = res.Switch0.Apower + res.Pm0.Apower + res.Em0.ActPower
		}
	}

	// Assure positive power response (Gen 1 EM devices can provide negative values)
	return math.Abs(power), nil
}

// Enabled implements the api.Charger interface
func (sh *Switch) Enabled() (bool, error) {
	switch sh.conn.gen {
	case 0, 1:
		var res Gen1SwitchResponse
		uri := fmt.Sprintf("%s/relay/%d", sh.conn.uri, sh.conn.channel)
		err := sh.conn.GetJSON(uri, &res)
		return res.Ison, err

	default:
		var res Gen2SwitchStatus
		err := sh.conn.execGen2Cmd("Switch.GetStatus", false, &res)
		return res.Output, err
	}
}

// Enable implements the api.Charger interface
func (sh *Switch) Enable(enable bool) error {
	var err error
	onoff := map[bool]string{true: "on", false: "off"}

	switch sh.conn.gen {
	case 0, 1:
		var res Gen1SwitchResponse
		uri := fmt.Sprintf("%s/relay/%d?turn=%s", sh.conn.uri, sh.conn.channel, onoff[enable])
		err = sh.conn.GetJSON(uri, &res)

	default:
		var res Gen2SwitchStatus
		err = sh.conn.execGen2Cmd("Switch.Set", enable, &res)
	}

	return err
}

// TotalEnergy implements the api.Meter interface
func (sh *Switch) TotalEnergy() (float64, error) {
	var energy float64

	switch sh.conn.gen {
	case 0, 1:
		var res Gen1Status
		uri := fmt.Sprintf("%s/status", sh.conn.uri)
		if err := sh.conn.GetJSON(uri, &res); err != nil {
			return 0, err
		}

		switch {
		case sh.conn.channel < len(res.Meters):
			energy = res.Meters[sh.conn.channel].Total
		case sh.conn.channel < len(res.EMeters):
			energy = res.EMeters[sh.conn.channel].Total
		default:
			return 0, errors.New("invalid channel, missing power meter")
		}

		energy = gen1Energy(sh.conn.model, energy)

	default:
		var res Gen2StatusResponse
		if err := sh.conn.execGen2Cmd("Shelly.GetStatus", false, &res); err != nil {
			return 0, err
		}

		switch sh.conn.channel {
		case 1:
			energy = res.Switch1.Aenergy.Total + res.Pm1.Aenergy.Total + res.Em1Data.TotalActEnergy - res.Em1Data.TotalActRetEnergy
		case 2:
			energy = res.Switch2.Aenergy.Total + res.Pm2.Aenergy.Total + res.Em2Data.TotalActEnergy - res.Em2Data.TotalActRetEnergy
		default:
			energy = res.Switch0.Aenergy.Total + res.Pm0.Aenergy.Total + res.Em0Data.TotalActEnergy - res.Em0Data.TotalActRetEnergy
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

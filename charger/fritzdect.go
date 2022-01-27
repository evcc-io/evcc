package charger

import (
	"encoding/xml"
	"errors"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/fritzdect"
)

// AVM FritzBox AHA interface specifications:
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf

// FritzDECT charger implementation
type FritzDECT struct {
	fritzdect    *fritzdect.Connection
	standbypower float64
	powerless    bool
}

func init() {
	registry.Add("fritzdect", NewFritzDECTFromConfig)
}

// NewFritzDECTFromConfig creates a fritzdect charger from generic config
func NewFritzDECTFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI          string
		AIN          string
		User         string
		Password     string
		StandbyPower float64
		Powerless    bool
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	return NewFritzDECT(cc.URI, cc.AIN, cc.User, cc.Password, cc.StandbyPower, cc.Powerless)
}

// NewFritzDECT creates a new connection with standbypower for charger
func NewFritzDECT(uri, ain, user, password string, standbypower float64, powerless bool) (*FritzDECT, error) {
	fritzdect, err := fritzdect.NewConnection(uri, ain, user, password)
	if err != nil {
		return nil, err
	}
	fd := &FritzDECT{
		fritzdect:    fritzdect,
		standbypower: standbypower,
		powerless:    powerless,
	}
	return fd, nil
}

// Status implements the api.Charger interface
func (c *FritzDECT) Status() (api.ChargeStatus, error) {
	// present 0/1 - DECT Switch connected to fritzbox (no/yes)
	var present bool
	var Ison bool
	var power float64

	resp, err := c.fritzdect.ExecCmd("getswitchpresent")
	if err == nil {
		present, err = strconv.ParseBool(resp)
		if err != nil {
			return api.StatusNone, err
		}
	}
	if !present {
		return api.StatusNone, api.ErrNotAvailable
	}

	if c.powerless {
		Ison, err = c.Enabled()
		if err != nil {
			return api.StatusNone, err
		}
	} else {
		power, err = c.fritzdect.CurrentPower()
		if err != nil {
			return api.StatusNone, err
		}
	}

	switch {
	// Charger status rules
	case !c.powerless && power <= c.standbypower:
		return api.StatusB, err
	case !c.powerless && power > c.standbypower:
		return api.StatusC, err
	// Simple powerless switch status rules
	case c.powerless && !Ison:
		return api.StatusB, err
	case c.powerless && Ison:
		return api.StatusC, err
	default:
		return api.StatusNone, api.ErrNotAvailable
	}
}

// Enabled implements the api.Charger interface
func (c *FritzDECT) Enabled() (bool, error) {
	// state 0/1 - DECT Switch state off/on (empty if unknown or error)
	resp, err := c.fritzdect.ExecCmd("getswitchstate")
	if err != nil {
		return false, err
	}

	if resp == "inval" {
		return false, api.ErrNotAvailable
	}

	return strconv.ParseBool(resp)
}

// Enable implements the api.Charger interface
func (c *FritzDECT) Enable(enable bool) error {
	cmd := "setswitchoff"
	if enable {
		cmd = "setswitchon"
	}

	// on 0/1 - DECT Switch state off/on (empty if unknown or error)
	resp, err := c.fritzdect.ExecCmd(cmd)

	var Ison bool
	if err == nil {
		Ison, err = strconv.ParseBool(resp)
	}

	switch {
	case err != nil:
		return err
	case enable && !Ison:
		return errors.New("switchOn failed")
	case !enable && Ison:
		return errors.New("switchOff failed")
	default:
		return nil
	}
}

// MaxCurrent implements the api.Charger interface
func (c *FritzDECT) MaxCurrent(current int64) error {
	return nil
}

var _ api.Meter = (*FritzDECT)(nil)

// CurrentPower implements the api.Meter interface
func (c *FritzDECT) CurrentPower() (float64, error) {
	power, err := c.fritzdect.CurrentPower()
	if power < c.standbypower || c.powerless {
		power = 0
	}

	return power, err
}

var _ api.ChargeRater = (*FritzDECT)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *FritzDECT) ChargedEnergy() (float64, error) {
	// fetch basicdevicestats
	resp, err := c.fritzdect.ExecCmd("getbasicdevicestats")
	if err != nil {
		return 0, err
	}

	// unmarshal devicestats
	var stats fritzdect.Devicestats
	if err = xml.Unmarshal([]byte(resp), &stats); err != nil {
		return 0, err
	}

	// select energy value of current day
	if len(stats.Energy.Values) == 0 {
		return 0, api.ErrNotAvailable
	}
	energylist := strings.Split(stats.Energy.Values[1], ",")
	energy, err := strconv.ParseFloat(energylist[0], 64)

	return energy / 1000, err
}

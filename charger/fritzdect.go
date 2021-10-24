package charger

import (
	"encoding/xml"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/fritzdect"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/fritzbox"
	"github.com/evcc-io/evcc/util/request"
)

// AVM FritzBox AHA interface specifications:
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf

// FritzDECT charger implementation
type fritzDECT struct {
	conn         *fritzbox.Connection
	standbypower float64
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
		SID          string
		StandbyPower float64
		Updated      time.Time
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		cc.URI = "https://fritz.box"
	}

	if cc.AIN == "" {
		return nil, errors.New("missing ain")
	}

	return NewFritzDECT(cc.URI, cc.AIN, cc.User, cc.Password, cc.SID, cc.StandbyPower, cc.Updated)
}

// NewFritzDECT creates FritzDECT charger
func NewFritzDECT(uri, ain, user, password, sid string, standbypower float64, updated time.Time) (*fritzDECT, error) {
	log := util.NewLogger("fritzdect")

	conn := &fritzbox.Connection{
		Helper:   request.NewHelper(log),
		URI:      strings.TrimRight(uri, "/"),
		AIN:      ain,
		User:     user,
		Password: password,
		SID:      sid,
	}

	c := &fritzDECT{
		conn:         conn,
		standbypower: standbypower,
	}

	c.conn.Client.Transport = request.NewTripper(log, request.InsecureTransport())

	return c, nil
}

// Status implements the api.Charger interface
func (c *fritzDECT) Status() (api.ChargeStatus, error) {
	// present 0/1 - DECT Switch connected to fritzbox (no/yes)
	var present int64
	resp, err := c.conn.ExecFritzDectCmd("getswitchpresent")
	if err == nil {
		present, err = strconv.ParseInt(resp, 10, 64)
		if err != nil {
			return api.StatusNone, err
		}
	}

	power, err := c.conn.CurrentPower()

	switch {
	case present == 1 && power <= c.standbypower:
		return api.StatusB, err
	case present == 1 && power > c.standbypower:
		return api.StatusC, err
	default:
		return api.StatusNone, api.ErrNotAvailable
	}
}

// Enabled implements the api.Charger interface
func (c *fritzDECT) Enabled() (bool, error) {
	// state 0/1 - DECT Switch state off/on (empty if unknown or error)
	resp, err := c.conn.ExecFritzDectCmd("getswitchstate")
	if err != nil {
		return false, err
	}

	if resp == "inval" {
		return false, api.ErrNotAvailable
	}

	state, err := strconv.ParseInt(resp, 10, 32)

	return state == 1, err
}

// Enable implements the api.Charger interface
func (c *fritzDECT) Enable(enable bool) error {
	cmd := "setswitchoff"
	if enable {
		cmd = "setswitchon"
	}

	// state 0/1 - DECT Switch state off/on (empty if unknown or error)
	resp, err := c.conn.ExecFritzDectCmd(cmd)

	var state int64
	if err == nil {
		state, err = strconv.ParseInt(resp, 10, 32)
	}

	switch {
	case err != nil:
		return err
	case enable && state == 0:
		return errors.New("switchOn failed")
	case !enable && state == 1:
		return errors.New("switchOff failed")
	default:
		return nil
	}
}

// MaxCurrent implements the api.Charger interface
func (c *fritzDECT) MaxCurrent(current int64) error {
	return nil
}

var _ api.Meter = (*fritzDECT)(nil)

// CurrentPower implements the api.Meter interface
func (c *fritzDECT) CurrentPower() (float64, error) {
	power, err := c.conn.CurrentPower()
	if power < c.standbypower {
		power = 0
	}

	return power, err
}

var _ api.ChargeRater = (*fritzDECT)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *fritzDECT) ChargedEnergy() (float64, error) {
	// fetch basicdevicestats
	resp, err := c.conn.ExecFritzDectCmd("getbasicdevicestats")
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

package charger

import (
	"errors"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tplink"
	"github.com/evcc-io/evcc/util"
)

// TPLink charger implementation
type TPLink struct {
	conn *tplink.Connection
	*switchSocket
}

func init() {
	registry.Add("tplink", NewTPLinkFromConfig)
}

// NewTPLinkFromConfig creates a TP-Link charger from generic config
func NewTPLinkFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		URI          string
		StandbyPower float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return NewTPLink(cc.URI, cc.StandbyPower)
}

// NewTPLink creates TP-Link charger
func NewTPLink(uri string, standbypower float64) (*TPLink, error) {
	conn, err := tplink.NewConnection(uri)
	if err != nil {
		return nil, err
	}

	c := &TPLink{
		conn: conn,
	}

	c.switchSocket = NewSwitchSocket(c.Enabled, c.conn.CurrentPower, standbypower)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *TPLink) Enabled() (bool, error) {
	var res tplink.SystemResponse
	if err := c.conn.ExecCmd(`{"system":{"get_sysinfo":null}}`, &res); err != nil {
		return false, err
	}

	if err := res.System.GetSysinfo.ErrCode; err != 0 {
		return false, fmt.Errorf("get_sysinfo error %d", err)
	}

	if !strings.Contains(res.System.GetSysinfo.Feature, "ENE") {
		return false, errors.New(res.System.GetSysinfo.Model + " not supported, energy meter feature missing")
	}

	return res.System.GetSysinfo.RelayState == 1, nil
}

// Enable implements the api.Charger interface
func (c *TPLink) Enable(enable bool) error {
	var res tplink.SystemResponse
	cmd := `{"system":{"set_relay_state":{"state":0}}}`
	if enable {
		cmd = `{"system":{"set_relay_state":{"state":1}}}`
	}

	if err := c.conn.ExecCmd(cmd, &res); err != nil {
		return err
	}

	if err := res.System.SetRelayState.ErrCode; err != 0 {
		return fmt.Errorf("set_relay_state error %d", err)
	}

	return nil
}

// MaxCurrent implements the api.Charger interface
func (c *TPLink) MaxCurrent(current int64) error {
	return nil
}

var _ api.MeterEnergy = (*TPLink)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *TPLink) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}

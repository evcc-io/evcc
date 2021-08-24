package charger

import (
	"errors"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Shelly api homepage
// https://shelly-api-docs.shelly.cloud/#common-http-api

// shellyRelayResponse provides the evcc shelly charger enabled information
type shellyRelayResponse struct {
	Ison bool `json:"ison,omitempty"`
}

// shellyStatusResponse provides the evcc shelly charger current power information
type shellyStatusResponse struct {
	Meters []struct {
		Power float64 `json:"power,omitempty"`
	} `json:"meters,omitempty"`
}

// Shelly charger implementation
type Shelly struct {
	*request.Helper
	uri          string
	channel      int
	standbypower float64
}

func init() {
	registry.Add("shelly", NewShellyFromConfig)
}

// NewShellyFromConfig creates a Shelly charger from generic config
func NewShellyFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI          string
		Channel      int
		StandbyPower float64
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return NewShelly(cc.URI, cc.Channel, cc.StandbyPower)
}

// NewShelly creates Shelly charger
func NewShelly(uri string, channel int, standbypower float64) (*Shelly, error) {
	c := &Shelly{
		Helper:       request.NewHelper(util.NewLogger("shelly")),
		uri:          strings.TrimRight(uri, "/"),
		channel:      channel,
		standbypower: standbypower,
	}

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *Shelly) Enabled() (bool, error) {
	var resp shellyRelayResponse
	err := c.GetJSON(fmt.Sprintf("%s/relay/%d", c.uri, c.channel), &resp)

	return resp.Ison, err
}

// Enable implements the api.Charger interface
func (c *Shelly) Enable(enable bool) error {
	var resp shellyRelayResponse
	onoff := map[bool]string{true: "on", false: "off"}
	err := c.GetJSON(fmt.Sprintf("%s/relay/%d?turn=%s", c.uri, c.channel, onoff[enable]), &resp)

	switch {
	case err != nil:
		return err
	case enable && !resp.Ison:
		return errors.New("switchOn failed")
	case !enable && resp.Ison:
		return errors.New("switchOff failed")
	default:
		return nil
	}
}

// MaxCurrent implements the api.Charger interface
func (c *Shelly) MaxCurrent(current int64) error {
	return nil
}

// Status implements the api.Charger interface
func (c *Shelly) Status() (api.ChargeStatus, error) {
	power, err := c.CurrentPower()
	if power > 0 {
		return api.StatusC, err
	}
	return api.StatusB, err
}

var _ api.Meter = (*Shelly)(nil)

// CurrentPower implements the api.Meter interface
func (c *Shelly) CurrentPower() (float64, error) {
	var resp shellyStatusResponse
	err := c.GetJSON(fmt.Sprintf("%s/%s", c.uri, "status"), &resp)

	if c.channel >= len(resp.Meters) {
		return 0, errors.New("invalid channel, power meter missing")
	}
	power := resp.Meters[c.channel].Power

	// ignore standby power
	if power < c.standbypower {
		power = 0
	}

	return power, err
}

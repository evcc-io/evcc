package charger

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Shelly api homepage
// https://shelly-api-docs.shelly.cloud/#common-http-api

// shellyshellyDeviceInfo provides the evcc shelly features and capabilities
type shellyDeviceInfo struct {
	Type      string `json:"type,omitempty"`
	Mac       string `json:"mac,omitempty"`
	Auth      bool   `json:"auth,omitempty"`
	NumMeters int64  `json:"num_meters,omitempty"`
}

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
	headers      map[string]string
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
		User         string
		Password     string
		Channel      int
		StandbyPower float64
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return NewShelly(cc.URI, cc.User, cc.Password, cc.Channel, cc.StandbyPower)
}

// NewShelly creates Shelly charger
func NewShelly(uri, user, password string, channel int, standbypower float64) (*Shelly, error) {
	u, err := url.Parse(uri)
	if err != nil || u.Host == "" {
		u.Host = uri
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}

	log := util.NewLogger("shelly")

	c := &Shelly{
		Helper:       request.NewHelper(log),
		headers:      make(map[string]string),
		channel:      channel,
		standbypower: standbypower,
	}

	var resp shellyDeviceInfo
	// All shellies from both Gen1 and Gen2 families expose the /shelly endpoint,
	// useful for discovery of devices and their features and capabilities.
	err = c.GetJSON(fmt.Sprintf("%s/shelly", u.String()), &resp)
	switch {
	case err != nil:
		return c, err
	case resp.NumMeters == 0:
		return c, fmt.Errorf("%s (%s) missing power meter", resp.Type, resp.Mac)
	case !resp.Auth:
		c.uri = u.String()
	case resp.Auth && (user == "" || password == ""):
		return c, fmt.Errorf("%s (%s) missing user/password", resp.Type, resp.Mac)
	default:
		c.uri = u.String()
		if err := provider.AuthHeaders(log, provider.Auth{
			Type:     "Basic",
			User:     user,
			Password: password,
		}, c.headers); err != nil {
			return c, err
		}

	}

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *Shelly) Enabled() (bool, error) {
	var resp shellyRelayResponse
	err := c.execCmd(fmt.Sprintf("%s/relay/%d", c.uri, c.channel), &resp)
	if err != nil {
		return false, err
	}

	return resp.Ison, err
}

// Enable implements the api.Charger interface
func (c *Shelly) Enable(enable bool) error {
	var resp shellyRelayResponse
	onoff := map[bool]string{true: "on", false: "off"}
	err := c.execCmd(fmt.Sprintf("%s/relay/%d?turn=%s", c.uri, c.channel, onoff[enable]), resp)

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
	err := c.execCmd(fmt.Sprintf("%s/%s", c.uri, "status"), &resp)
	if err != nil {
		return 0, err
	}

	if c.channel >= len(resp.Meters) {
		return 0, errors.New("invalid channel, missing power meter")
	}
	power := resp.Meters[c.channel].Power

	// ignore standby power
	if power < c.standbypower {
		power = 0
	}

	return power, err
}

// execCmd executes a shelly api gen1/gen2 command and provides the response
func (c *Shelly) execCmd(cmd string, res interface{}) error {
	req, err := request.New(http.MethodGet, cmd, nil, c.headers)
	if err != nil {
		return err
	}
	return c.DoJSON(req, res)
}

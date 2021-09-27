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
	Gen       int    `json:"gen,omitempty"`
	Id        string `json:"id,omitempty"`
	Model     string `json:"model,omitempty"`
	Type      string `json:"type,omitempty"`
	Mac       string `json:"mac,omitempty"`
	Auth      bool   `json:"auth,omitempty"`
	AuthEn    bool   `json:"auth_en,omitempty"`
	NumMeters int    `json:"num_meters,omitempty"`
}

// shellyRelayResponse provides the evcc shelly charger enabled information
type switchGetStatusResponse struct {
	// Shelly Gen1 relay response
	Ison bool `json:"ison,omitempty"`
	// Shelly Gen2 Switch.GetStatus response
	Output bool `json:"output,omitempty"`
}

// shellyStatusResponse provides the evcc shelly charger current power information
type shellyGetStatusResponse struct {
	// Shelly Gen1 status response
	Meters []struct {
		Power float64 `json:"power,omitempty"`
	} `json:"meters,omitempty"`
	// Shelly Gen2 Get.Status response
	Switch0 struct {
		Apower float64 `json:"apower,omitempty"`
	} `json:"switch:0,omitempty"`
	Switch1 struct {
		Apower float64 `json:"apower,omitempty"`
	} `json:"switch:1,omitempty"`
	Switch2 struct {
		Apower float64 `json:"apower,omitempty"`
	} `json:"switch:2,omitempty"`
}

// Shelly charger implementation
type Shelly struct {
	*request.Helper
	uri          string
	gen          int // Shelly api generation
	authon       bool
	user         string
	password     string
	ah1          map[string]string
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
		u.Scheme = "http"
	}

	log := util.NewLogger("shelly")
	c := &Shelly{
		Helper:       request.NewHelper(log),
		gen:          1,
		user:         user,
		password:     password,
		channel:      channel,
		standbypower: standbypower,
	}

	var resp shellyDeviceInfo
	// All shellies from both Gen1 and Gen2 families expose the /shelly endpoint,
	// useful for discovery of devices and their features and capabilities.
	err = c.GetJSON(fmt.Sprintf("%s/shelly", u.String()), &resp)
	if err != nil {
		return c, err
	}

	if resp.Gen == 0 {
		resp.Gen = 1
	}
	c.gen = resp.Gen

	if resp.Model == "" {
		resp.Model = resp.Type
	}

	switch {
	case c.gen == 1:
		// Shelly GEN 1 API
		// https://shelly-api-docs.shelly.cloud/gen1/#shelly-family-overview
		u.Scheme = "http"
		switch {
		case resp.NumMeters == 0:
			return c, fmt.Errorf("%s (%s) gen1 missing power meter ", resp.Model, resp.Mac)
		case resp.Auth && (user == "" || password == ""):
			return c, fmt.Errorf("%s (%s) missing user/password", resp.Model, resp.Mac)
		default:
			c.uri = u.String()
		}
	case c.gen == 2:
		// Shelly GEN 2 API
		// https://shelly-api-docs.shelly.cloud/gen2/
		switch {
		case resp.AuthEn && (user == "" || password == ""):
			return c, fmt.Errorf("%s (%s) missing user/password", resp.Model, resp.Mac)
		default:
			if u.Scheme == "https" {
				c.Client.Transport = request.NewTripper(log, request.InsecureTransport())
			}
			c.uri = u.String() + "/rpc"
		}
	default:
		return c, fmt.Errorf("%s (%s) unknown api generation (%d)", resp.Type, resp.Model, resp.Gen)
	}

	c.authon = resp.Auth || resp.AuthEn

	if c.gen == 1 && c.authon {
		ah1 := make(map[string]string)
		if err := provider.AuthHeaders(log, provider.Auth{
			Type:     "Basic",
			User:     user,
			Password: password,
		}, ah1); err != nil {
			return c, err
		}
		c.ah1 = ah1
	}

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *Shelly) Enabled() (bool, error) {
	var resp switchGetStatusResponse
	cmd := map[int]string{1: "%s/relay/%d", 2: "%s/Switch.GetStatus?id=%d"}
	err := c.execCmd(fmt.Sprintf(cmd[c.gen], c.uri, c.channel), &resp)
	if err != nil {
		return false, err
	}
	if c.gen == 1 {
		return resp.Ison, err
	}
	return resp.Output, err
}

// Enable implements the api.Charger interface
func (c *Shelly) Enable(enable bool) error {
	var err error
	var resp switchGetStatusResponse
	cmd := map[int]string{1: "%s/relay/%d?turn=%s", 2: "%s/Switch.Set?id=%d&on=%t"}

	switch c.gen {
	case 1:
		onoff := map[bool]string{true: "on", false: "off"}
		err = c.execCmd(fmt.Sprintf(cmd[c.gen], c.uri, c.channel, onoff[enable]), resp)
	default:
		err = c.execCmd(fmt.Sprintf(cmd[c.gen], c.uri, c.channel, enable), resp)
		if err != nil {
			return err
		}
		resp.Ison, err = c.Enabled()
	}

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
	var resp shellyGetStatusResponse
	cmd := map[int]string{1: "status", 2: "Shelly.GetStatus"}
	err := c.execCmd(fmt.Sprintf("%s/%s", c.uri, cmd[c.gen]), &resp)
	if err != nil {
		return 0, err
	}
	var power float64
	if c.gen == 1 {
		if c.channel >= len(resp.Meters) {
			return 0, errors.New("invalid channel, missing power meter")
		}
		power = resp.Meters[c.channel].Power
	}
	if c.gen == 2 {
		switch c.channel {
		case 1:
			power = resp.Switch1.Apower
		case 2:
			power = resp.Switch2.Apower
		default:
			power = resp.Switch0.Apower
		}
	}
	// ignore standby power
	if power < c.standbypower {
		power = 0
	}
	return power, err
}

// execCmd executes a shelly api gen1/gen2 command and provides the response
func (c *Shelly) execCmd(cmd string, res interface{}) error {
	req, err := request.New(http.MethodGet, cmd, nil, c.ah1)
	if err != nil {
		return err
	}
	return c.DoJSON(req, res)
}

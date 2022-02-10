package charger

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/shelly"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

// Shelly charger implementation
type Shelly struct {
	*request.Helper
	log          *util.Logger
	uri          string
	gen          int // Shelly api generation
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
	for _, suffix := range []string{"/", "/rcp", "/shelly"} {
		uri = strings.TrimSuffix(uri, suffix)
	}

	log := util.NewLogger("shelly")
	client := request.NewHelper(log)

	// Shelly Gen1 and Gen2 families expose the /shelly endpoint
	var resp shelly.DeviceInfo
	if err := client.GetJSON(fmt.Sprintf("%s/shelly", util.DefaultScheme(uri, "http")), &resp); err != nil {
		return nil, err
	}

	c := &Shelly{
		Helper:       client,
		log:          log,
		channel:      channel,
		standbypower: standbypower,
		gen:          resp.Gen,
	}

	c.Client.Transport = request.NewTripper(log, transport.Insecure())

	if (resp.Auth || resp.AuthEn) && (user == "" || password == "") {
		return c, fmt.Errorf("%s (%s) missing user/password", resp.Model, resp.Mac)
	}

	switch c.gen {
	case 0, 1:
		// Shelly GEN 1 API
		// https://shelly-api-docs.shelly.cloud/gen1/#shelly-family-overview
		c.uri = util.DefaultScheme(uri, "http")
		if user != "" {
			log.Redact(transport.BasicAuthHeader(user, password))
			c.Client.Transport = transport.BasicAuth(user, password, c.Client.Transport)
		}

		if resp.NumMeters == 0 {
			// Shelly1 force static mode with fake power http://192.168.178.xxx/settings/power/0?power=standbypower+1
			uri := fmt.Sprintf("%s/settings/power/%d?power=%d", c.uri, c.channel, int(math.Abs(c.standbypower)+1))
			if err := c.GetJSON(uri, &resp); err != nil {
				return c, err
			}
		}

	case 2:
		// Shelly GEN 2 API
		// https://shelly-api-docs.shelly.cloud/gen2/
		c.uri = fmt.Sprintf("%s/rpc", util.DefaultScheme(uri, "http"))
		if user != "" {
			c.Client.Transport = digest.NewTransport(user, password, c.Client.Transport)
		}

	default:
		return c, fmt.Errorf("%s (%s) unknown api generation (%d)", resp.Type, resp.Model, c.gen)
	}

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *Shelly) Enabled() (bool, error) {
	switch c.gen {
	case 0, 1:
		var resp shelly.Gen1SwitchResponse
		uri := fmt.Sprintf("%s/relay/%d", c.uri, c.channel)
		err := c.GetJSON(uri, &resp)
		return resp.Ison, err

	default:
		var resp shelly.Gen2SwitchResponse
		err := c.execGen2Cmd("Switch.GetStatus", false, &resp)
		return resp.Output, err
	}
}

// Enable implements the api.Charger interface
func (c *Shelly) Enable(enable bool) error {
	var err error
	onoff := map[bool]string{true: "on", false: "off"}

	switch c.gen {
	case 0, 1:
		var resp shelly.Gen1SwitchResponse
		uri := fmt.Sprintf("%s/relay/%d?turn=%s", c.uri, c.channel, onoff[enable])
		err = c.GetJSON(uri, &resp)

	default:
		var resp shelly.Gen2SwitchResponse
		err = c.execGen2Cmd("Switch.Set", enable, &resp)
	}

	if err != nil {
		return err
	}

	enabled, err := c.Enabled()
	switch {
	case err != nil:
		return err
	case enable != enabled:
		return fmt.Errorf("switch %s failed", onoff[enable])
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
	res := api.StatusB

	// static mode
	if c.standbypower < 0 {
		on, err := c.Enabled()
		if on {
			res = api.StatusC
		}

		return res, err
	}

	// standby power mode
	power, err := c.CurrentPower()
	if power > c.standbypower {
		res = api.StatusC
	}

	return res, err
}

var _ api.Meter = (*Shelly)(nil)

// CurrentPower implements the api.Meter interface
func (c *Shelly) CurrentPower() (float64, error) {
	var power float64
	switch c.gen {
	case 0, 1:
		var resp shelly.Gen1StatusResponse
		uri := fmt.Sprintf("%s/status", c.uri)
		if err := c.GetJSON(uri, &resp); err != nil {
			return 0, err
		}

		if c.channel >= len(resp.Meters) {
			return 0, errors.New("invalid channel, missing power meter")
		}

		power = resp.Meters[c.channel].Power

	default:
		var resp shelly.Gen2StatusResponse
		if err := c.execGen2Cmd("Shelly.GetStatus", false, &resp); err != nil {
			return 0, err
		}

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
	if power <= c.standbypower {
		power = 0
	}

	return power, nil
}

// execGen2Cmd executes a shelly api gen1/gen2 command and provides the response
func (c *Shelly) execGen2Cmd(method string, enable bool, res interface{}) error {
	// Shelly gen 2 rfc7616 authentication
	// https://shelly-api-docs.shelly.cloud/gen2/Overview/CommonDeviceTraits#authentication
	// https://datatracker.ietf.org/doc/html/rfc7616

	data := &shelly.Gen2RpcPost{
		Id:     c.channel,
		On:     enable,
		Src:    "evcc",
		Method: method,
	}

	req, err := request.New(http.MethodPost, fmt.Sprintf("%s/%s", c.uri, method), request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	return c.DoJSON(req, &res)
}

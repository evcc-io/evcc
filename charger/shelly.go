package charger

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/shelly"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Shelly charger implementation
type Shelly struct {
	*request.Helper
	log          *util.Logger
	uri          string
	gen          int // Shelly api generation
	authRequired bool
	user         string
	password     string
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
		user:         user,
		password:     password,
		channel:      channel,
		standbypower: standbypower,
		gen:          resp.Gen,
		authRequired: resp.Auth || resp.AuthEn,
	}

	c.Client.Transport = request.NewTripper(log, request.InsecureTransport())

	if c.authRequired && (user == "" || password == "") {
		return c, fmt.Errorf("%s (%s) missing user/password", resp.Model, resp.Mac)
	}

	switch c.gen {
	case 0, 1:
		// Shelly GEN 1 API
		// https://shelly-api-docs.shelly.cloud/gen1/#shelly-family-overview
		c.uri = util.DefaultScheme(uri, "http")
		if resp.NumMeters == 0 {
			return c, fmt.Errorf("%s (%s) gen1 missing power meter ", resp.Model, resp.Mac)
		}
	case 2:
		// Shelly GEN 2 API
		// https://shelly-api-docs.shelly.cloud/gen2/
		c.uri = fmt.Sprintf("%s/rpc", util.DefaultScheme(uri, "https"))
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
		cmd := fmt.Sprintf("%s/relay/%d", c.uri, c.channel)
		err := c.execGen1Cmd(cmd, &resp)
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
		cmd := fmt.Sprintf("%s/relay/%d?turn=%s", c.uri, c.channel, onoff[enable])
		err = c.execGen1Cmd(cmd, &resp)

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
	power, err := c.CurrentPower()
	if power > c.standbypower {
		return api.StatusC, err
	}

	return api.StatusB, err
}

var _ api.Meter = (*Shelly)(nil)

// CurrentPower implements the api.Meter interface
func (c *Shelly) CurrentPower() (float64, error) {
	var power float64
	switch c.gen {
	case 0, 1:
		var resp shelly.Gen1StatusResponse
		cmd := fmt.Sprintf("%s/status", c.uri)
		if err := c.execGen1Cmd(cmd, &resp); err != nil {
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

// executes a shelly gen1 command and provides the response
func (c *Shelly) execGen1Cmd(cmd string, res interface{}) error {
	hab := make(map[string]string)
	// Shelly gen 1 basic authentication
	// https://shelly-api-docs.shelly.cloud/gen1/#http-dialect
	if c.authRequired {
		if err := provider.AuthHeaders(c.log, provider.Auth{
			Type:     "Basic",
			User:     c.user,
			Password: c.password,
		}, hab); err != nil {
			return err
		}
	}

	req, err := request.New(http.MethodGet, cmd, nil, hab)
	if err != nil {
		return err
	}

	return c.DoJSON(req, &res)
}

// execGen2Cmd executes a shelly api gen1/gen2 command and provides the response
func (c *Shelly) execGen2Cmd(method string, enable bool, res interface{}) error {
	// Shelly gen 2 rfc7616 authentication
	// https://shelly-api-docs.shelly.cloud/gen2/Overview/CommonDeviceTraits#authentication
	// https://datatracker.ietf.org/doc/html/rfc7616
	// post

	postjson := &shelly.Gen2RpcPost{
		Id:     c.channel,
		On:     enable,
		Src:    "evcc",
		Method: method,
	}

	req, err := request.New(http.MethodPost, fmt.Sprintf("%s/%s", c.uri, method), request.MarshalJSON(postjson), request.JSONEncoding)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		var authorization map[string]string = digestAuthParams(resp)
		// a1
		a1 := fmt.Sprintf("%s:%s:%s", c.user, authorization["realm"], c.password)
		h := sha256.New()
		h.Reset()
		fmt.Fprint(h, a1)
		ha1 := hex.EncodeToString(h.Sum(nil))
		// a2
		a2 := fmt.Sprintf("%s:%s", req.Method, req.URL.RequestURI())
		h.Reset()
		fmt.Fprint(h, a2)
		ha2 := hex.EncodeToString(h.Sum(nil))

		// response
		cnonce := randCnonce()
		response := strings.Join([]string{ha1, authorization["nonce"], "00000001" /* nc */, cnonce, authorization["qop"], ha2}, ":")
		h.Reset()
		fmt.Fprint(h, response)
		response = hex.EncodeToString(h.Sum(nil))

		// auth header
		AuthHeader := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s", qop=%s, nc=%s, cnonce="%s", opaque="%s", algorithm="%s"`,
			c.user, authorization["realm"], authorization["nonce"], req.URL.RequestURI(), response, authorization["qop"], "00000001" /* nc */, cnonce, authorization["opaque"], authorization["algorithm"])

		req, err = request.New(http.MethodPost, fmt.Sprintf("%s/%s", c.uri, method), request.MarshalJSON(postjson), request.JSONEncoding)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", AuthHeader)

		resp, err = c.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return json.Unmarshal(bodyBytes, &res)
	}

	return fmt.Errorf("unhandeled hhtp status code: %d", resp.StatusCode)
}

// parse Shelly authorization header
func digestAuthParams(r *http.Response) map[string]string {
	s := strings.SplitN(r.Header.Get("Www-Authenticate"), " ", 2)
	if len(s) != 2 || s[0] != "Digest" {
		return nil
	}

	result := map[string]string{}
	for _, kv := range strings.Split(s[1], ",") {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		result[strings.Trim(parts[0], "\" ")] = strings.Trim(parts[1], "\" ")
	}
	return result
}

// create random client nonce
func randCnonce() string {
	b := make([]byte, 16)
	io.ReadFull(rand.Reader, b)
	return hex.EncodeToString(b)
}

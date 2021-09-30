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
	"net/url"
	"strings"

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
	log          *util.Logger
	uri          string
	gen          int // Shelly api generation
	authon       bool
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
		log:          log,
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

	c.authon = resp.Auth || resp.AuthEn
	if c.authon && (user == "" || password == "") {
		return c, fmt.Errorf("%s (%s) missing user/password", resp.Model, resp.Mac)
	}

	if resp.Model == "" {
		resp.Model = resp.Type
	}

	switch {
	case c.gen == 1:
		// Shelly GEN 1 API
		// https://shelly-api-docs.shelly.cloud/gen1/#shelly-family-overview
		u.Scheme = "http"
		c.uri = u.String()
		if resp.NumMeters == 0 {
			return c, fmt.Errorf("%s (%s) gen1 missing power meter ", resp.Model, resp.Mac)
		}
	case c.gen == 2:
		// Shelly GEN 2 API
		// https://shelly-api-docs.shelly.cloud/gen2/
		if u.Scheme == "https" {
			c.Client.Transport = request.NewTripper(log, request.InsecureTransport())
		}
		c.uri = u.String() + "/rpc"
	default:
		return c, fmt.Errorf("%s (%s) unknown api generation (%d)", resp.Type, resp.Model, resp.Gen)
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

	if c.gen == 1 || (c.gen == 2 && !c.authon) {
		hab := make(map[string]string)
		if c.gen == 1 && c.authon {
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
		return c.DoJSON(req, res)
	}

	if c.gen == 2 && c.authon {

		req, err := request.New(http.MethodPost, cmd, nil, request.URLEncoding)
		if err != nil {
			return err
		}
		c.log.TRACE.Printf("cmd: %s", cmd)

		resp, err := c.Do(req)
		if err != nil {
			return err
		}
		// read the whole body and then close it to reuse the http connection
		// otherwise it *could* fail in certain environments (behind proxy for instance)
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			var authorization map[string]string = digestAuthParams(resp)
			// a1
			a1 := fmt.Sprintf("%s:%s:%s", c.user, authorization["realm"], c.password)
			c.log.TRACE.Printf("a1: %s", a1)
			h := sha256.New()
			h.Reset()
			fmt.Fprint(h, a1)
			ha1 := hex.EncodeToString(h.Sum(nil))
			// a2
			a2 := fmt.Sprintf("%s:%s", req.Method, req.URL.RequestURI())
			c.log.TRACE.Printf("a2: %s", a2)
			h.Reset()
			fmt.Fprint(h, a2)
			ha2 := hex.EncodeToString(h.Sum(nil))

			// response
			cnonce := randCnonce()
			response := strings.Join([]string{ha1, authorization["nonce"], "00000001" /* nc */, cnonce, authorization["qop"], ha2}, ":")
			c.log.TRACE.Printf("response: %s", response)
			h.Reset()
			fmt.Fprint(h, response)
			response = hex.EncodeToString(h.Sum(nil))

			// auth header
			AuthHeader := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s", qop=%s, nc=%s, cnonce="%s", opaque="%s", algorithm="%s"`,
				c.user, authorization["realm"], authorization["nonce"], req.URL.RequestURI(), response, authorization["qop"], "00000001" /* nc */, cnonce, authorization["opaque"], authorization["algorithm"])

			// post json
			data := struct {
				Id     string `json:"id"`
				Src    string `json:"src"`
				Method string `json:"method"`
			}{
				Id:     fmt.Sprint(c.channel),
				Src:    "evcc",
				Method: req.URL.RequestURI(),
			}
			req, err = request.New(http.MethodPost, cmd, request.MarshalJSON(data), request.JSONEncoding)
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", AuthHeader)

			resp, err = c.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				bodyBytes, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				return json.Unmarshal(bodyBytes, res)
			} else {
				return fmt.Errorf("unknown auth status code: %d", resp.StatusCode)
			}
		}
	}
	return nil
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

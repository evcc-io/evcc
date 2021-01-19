package charger

import (
	"fmt"
	"net/http"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger/easee"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

// https://api.easee.cloud/index.html

const easyAPI = "https://api.easee.cloud/api"

// EaseeToken is the /api/accounts/token and /api/accounts/refresh_token response
type EaseeToken struct {
	AccessToken  string    `json:"accessToken"`
	ExpiresIn    float32   `json:"expiresIn"`
	TokenType    string    `json:"tokenType"`
	RefreshToken string    `json:"refreshToken"`
	Valid        time.Time // helper to store validity timestamp
}

// Easee charger implementation
type Easee struct {
	*request.Helper
	token         EaseeToken
	charger       string
	site, circuit int
	status        easee.ChargerStatus
	updated       time.Time
	cache         time.Duration
}

func init() {
	registry.Add("easee", NewEaseeFromConfig)
}

// NewEaseeFromConfig creates a go-e charger from generic config
func NewEaseeFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		User     string
		Password string
		Charger  string
		// Site     string
		// Circuit  int
		Cache time.Duration
	}{
		Cache: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEasee(cc.User, cc.Password, cc.Charger, cc.Cache)
}

// NewEasee creates Easee charger
func NewEasee(user, password, charger string, cache time.Duration) (*Easee, error) {
	c := &Easee{
		Helper:  request.NewHelper(util.NewLogger("easee")),
		charger: charger,
		cache:   cache,
	}

	err := c.login(user, password)
	if err != nil {
		return c, err
	}

	// find charger
	if charger == "" {
		chargers, err := c.chargers()
		if err != nil {
			return c, err
		}

		if len(chargers) != 1 {
			return c, fmt.Errorf("cannot determine charger id, found: %v", chargers)
		}

		c.charger = chargers[0].ID
	}

	// find site and circuit
	site, err := c.chargerDetails()
	if err != nil {
		return c, err
	}

	if len(site.Circuits) != 1 {
		return c, fmt.Errorf("cannot determine circuit id, found: %v", site.Circuits)
	}

	c.site = site.ID
	c.circuit = site.Circuits[0].ID

	return c, err
}

func (c *Easee) login(user, password string) error {
	data := struct {
		Username string `json:"userName"`
		Password string `json:"password"`
	}{
		Username: user,
		Password: password,
	}

	uri := fmt.Sprintf("%s%s", easyAPI, "/accounts/token")
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	if err == nil {
		err = c.DoJSON(req, &c.token)
		c.token.Valid = time.Now().Add(time.Second * time.Duration(c.token.ExpiresIn))
	}

	return err
}

func (c *Easee) refreshToken() error {
	data := struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}{
		AccessToken:  c.token.AccessToken,
		RefreshToken: c.token.RefreshToken,
	}

	uri := fmt.Sprintf("%s%s", easyAPI, "/accounts/refresh_token")
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	var token EaseeToken
	if err == nil {
		err = c.DoJSON(req, &token)
		token.Valid = time.Now().Add(time.Second * time.Duration(token.ExpiresIn))
	}

	if err == nil {
		c.token = token
	}

	return err
}

// request creates JSON HTTP request with valid access token
func (c *Easee) request(method, path string, body interface{}) (*http.Request, error) {
	if c.token.Valid.After(time.Now()) {
		if err := c.refreshToken(); err != nil {
			return nil, err
		}
	}

	uri := fmt.Sprintf("%s%s", easyAPI, path)

	req, err := request.New(method, uri, request.MarshalJSON(body), request.JSONEncoding)
	if err == nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token.AccessToken))
	}

	return req, err
}

func (c *Easee) chargers() (res []easee.Charger, err error) {
	req, err := c.request(http.MethodGet, "/chargers", nil)
	if err != nil {
		return nil, err
	}

	err = c.DoJSON(req, &res)
	return res, err
}

func (c *Easee) chargerDetails() (res easee.Site, err error) {
	req, err := c.request(http.MethodGet, fmt.Sprintf("/chargers/%s/site", c.charger), nil)
	if err != nil {
		return res, err
	}

	err = c.DoJSON(req, &res)
	return res, err
}

func (c *Easee) state() (easee.ChargerStatus, error) {
	if time.Since(c.updated) < c.cache {
		return c.status, nil
	}

	req, err := c.request(http.MethodGet, fmt.Sprintf("/chargers/%s/state", c.charger), nil)
	if err == nil {
		if err = c.DoJSON(req, &c.status); err == nil {
			c.updated = time.Now()
		}
	}

	return c.status, err
}

// Status implements the Charger.Status interface
func (c *Easee) Status() (api.ChargeStatus, error) {
	res, err := c.state()
	if err != nil {
		return api.StatusNone, err
	}

	switch res.ChargerOpMode {
	case 1:
		return api.StatusA, nil
	case 2, 4, 6:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	case 5:
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown opmode: %d", res.ChargerOpMode)
	}
}

// Enabled implements the Charger.Enabled interface
func (c *Easee) Enabled() (bool, error) {
	res, err := c.state()
	return res.IsOnline, err
}

// Enable implements the Charger.Enable interface
func (c *Easee) Enable(enable bool) error {
	data := easee.ChargerSettings{
		Enabled: &enable,
	}

	req, err := c.request(http.MethodPost, fmt.Sprintf("/chargers/%s/settings", c.charger), data)
	if err == nil {
		_, err = c.Do(req)
		c.updated = time.Time{} // clear cache
	}

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (c *Easee) MaxCurrent(current int64) error {
	cur := int(current)
	data := easee.CircuitSettings{
		DynamicCircuitCurrentP1: &cur,
		DynamicCircuitCurrentP2: &cur,
		DynamicCircuitCurrentP3: &cur,
	}

	req, err := c.request(http.MethodPost, fmt.Sprintf("/sites/%d/circuits/%d/settings", c.site, c.circuit), data)
	if err == nil {
		_, err = c.Do(req)
		c.updated = time.Time{} // clear cache
	}

	return err
}

// CurrentPower implements the Meter interface.
func (c *Easee) CurrentPower() (float64, error) {
	res, err := c.state()
	return 1e3 * float64(res.TotalPower), err
}

// ChargedEnergy implements the ChargeRater interface
func (c *Easee) ChargedEnergy() (float64, error) {
	res, err := c.state()
	return float64(res.SessionEnergy), err
}

// Currents implements the MeterCurrent interface
func (c *Easee) Currents() (float64, float64, float64, error) {
	res, err := c.state()
	return float64(res.CircuitTotalPhaseConductorCurrentL1),
		float64(res.CircuitTotalPhaseConductorCurrentL2),
		float64(res.CircuitTotalPhaseConductorCurrentL3),
		err
}

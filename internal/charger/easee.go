package charger

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/charger/easee"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/util/sponsor"
)

// Easee charger implementation
type Easee struct {
	*request.Helper
	*easee.Identity
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
		Cache    time.Duration
	}{
		Cache: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEasee(cc.User, cc.Password, cc.Charger, cc.Cache)
}

// NewEasee creates Easee charger
func NewEasee(user, password, charger string, cache time.Duration) (*Easee, error) {
	log := util.NewLogger("easee")

	if !sponsor.IsAuthorized() {
		return nil, errors.New("easee requires evcc sponsorship, register at https://cloud.evcc.io")
	}

	c := &Easee{
		Helper:  request.NewHelper(log),
		charger: charger,
		cache:   cache,
	}

	var err error
	if c.Identity, err = easee.NewIdentity(log, user, password); err != nil {
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

// request creates JSON HTTP request with valid access token
func (c *Easee) request(method, path string, body interface{}) (*http.Request, error) {
	uri := fmt.Sprintf("%s%s", easee.API, path)

	req, err := request.New(method, uri, request.MarshalJSON(body), request.JSONEncoding)
	if err == nil {
		var token string
		token, err = c.Token()
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
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

// Status implements the api.Charger interface
func (c *Easee) Status() (api.ChargeStatus, error) {
	res, err := c.state()
	if err != nil {
		return api.StatusNone, err
	}

	switch res.ChargerOpMode {
	case easee.ModeDisconnected:
		return api.StatusA, nil
	case easee.ModeAwaitingStart, easee.ModeCompleted, easee.ModeReadyToCharge:
		return api.StatusB, nil
	case easee.ModeCharging:
		return api.StatusC, nil
	case easee.ModeError:
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown opmode: %d", res.ChargerOpMode)
	}
}

// Enabled implements the api.Charger interface
func (c *Easee) Enabled() (bool, error) {
	res, err := c.state()
	return res.ChargerOpMode == easee.ModeCharging || res.ChargerOpMode == easee.ModeReadyToCharge, err
}

// Enable implements the api.Charger interface
func (c *Easee) Enable(enable bool) error {
	res, err := c.state()
	if err != nil {
		return err
	}

	// enable charger once
	if enable && !res.IsOnline {
		data := easee.ChargerSettings{
			Enabled: &enable,
		}

		var req *http.Request
		if req, err = c.request(http.MethodPost, fmt.Sprintf("/chargers/%s/settings", c.charger), data); err == nil {
			_, err = c.Do(req)
			c.updated = time.Time{} // clear cache
		}

		return err
	}

	// resume/stop charger
	action := "pause_charging"
	if enable {
		action = "resume_charging"
	}

	var req *http.Request
	if req, err = c.request(http.MethodPost, fmt.Sprintf("/chargers/%s/commands/%s", c.charger, action), nil); err == nil {
		_, err = c.Do(req)
		c.updated = time.Time{} // clear cache
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *Easee) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Easee)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *Easee) MaxCurrentMillis(current float64) error {
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

var _ api.Meter = (*Easee)(nil)

// CurrentPower implements the api.Meter interface.
func (c *Easee) CurrentPower() (float64, error) {
	res, err := c.state()
	return 1e3 * res.TotalPower, err
}

var _ api.ChargeRater = (*Easee)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *Easee) ChargedEnergy() (float64, error) {
	res, err := c.state()
	return res.SessionEnergy, err
}

var _ api.MeterCurrent = (*Easee)(nil)

// Currents implements the api.MeterCurrent interface
func (c *Easee) Currents() (float64, float64, float64, error) {
	res, err := c.state()
	return res.CircuitTotalPhaseConductorCurrentL1,
		res.CircuitTotalPhaseConductorCurrentL2,
		res.CircuitTotalPhaseConductorCurrentL3,
		err
}

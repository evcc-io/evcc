package charger

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openevse"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// OpenEVSE charger implementation
type OpenEVSE struct {
	*request.Helper
	uri     string
	statusG provider.Cacheable[openevse.Status]
	current int
	enabled bool
}

func init() {
	registry.Add("openevse", NewOpenEVSEFromConfig)
}

// go:generate go run ../cmd/tools/decorate.go -f decorateOpenEVSE -b ""*OpenEVSE" -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error"

// NewOpenEVSEFromConfig creates an OpenEVSE charger from generic config
func NewOpenEVSEFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI      string
		User     string
		Password string
		Cache    time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return NewOpenEVSE(cc.URI, cc.User, cc.Password, cc.Cache)
}

// NewOpenEVSE creates OpenEVSE charger
func NewOpenEVSE(uri, user, password string, cache time.Duration) (api.Charger, error) {
	basicAuth := transport.BasicAuthHeader(user, password)
	log := util.NewLogger("openevse").Redact(user, password, basicAuth)

	c := &OpenEVSE{
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
	}

	if user != "" && password != "" {
		c.Client.Transport = transport.BasicAuth(user, password, c.Client.Transport)
	}

	c.statusG = provider.ResettableCached(func() (openevse.Status, error) {
		var res openevse.Status

		uri := fmt.Sprintf("%s/status", c.uri)
		err := c.GetJSON(uri, &res)

		return res, err
	}, cache)

	var phases1p3p func(int) error
	if err := c.hasPhaseSwitchCapabilities(); err == nil {
		phases1p3p = c.phases1p3p

		// disable EVSE's own 1/3-phase auto-switching
		if err := c.rapiCommand("$S8 0"); err != nil {
			return c, err
		}
	}

	return decorateOpenEVSE(c, phases1p3p), nil
}

func (c *OpenEVSE) setOverride() error {
	var data openevse.Override
	uri := fmt.Sprintf("%s/override", c.uri)

	if err := c.GetJSON(uri, &data); err != nil {
		return err
	}

	state := openevse.Disabled
	if c.enabled {
		state = openevse.Enabled
	}
	data.State = &state
	data.MaxCurrent = &c.current

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		_, err = c.DoBody(req)
	}

	return err
}

func (c *OpenEVSE) rapiCommand(command string) error {
	var res struct {
		Cmd, Ret string
	}

	uri := fmt.Sprintf("%s/r?json=1&rapi=%s", c.uri, url.QueryEscape(command))

	err := c.GetJSON(uri, &res)
	if err == nil && !strings.HasPrefix(res.Ret, "$OK") {
		err = fmt.Errorf("rapi command failed: %s", res.Ret)
	}

	return err
}

func (c *OpenEVSE) hasPhaseSwitchCapabilities() error {
	return c.rapiCommand("$G7")
}

// Status implements the api.Charger interface
func (c *OpenEVSE) Status() (api.ChargeStatus, error) {
	res, err := c.statusG.Get()
	/*
		0: "unknown",
		1: "not connected",
		2: "connected",
		3: "charging",
		4: "vent required",
		5: "diode check failed",
		6: "gfci fault",
		7: "no ground",
		8: "stuck relay",
		9: "gfci self-test failure",
		10: "over temperature",
		11: "over current",
		254: "sleeping",
		255: "disabled"
	*/

	switch state := *res.State; state {
	case 1:
		return api.StatusA, err
	case 2, 254, 255:
		if *res.Vehicle == 1 {
			return api.StatusB, err
		}
		return api.StatusA, err
	case 3:
		return api.StatusC, err
	case 4:
		return api.StatusD, err
	case 5, 6, 7, 8, 9, 10, 11:
		return api.StatusF, err
	default:
		return api.StatusNone, fmt.Errorf("unknown EVSE state: %d", state)
	}
}

// Enabled implements the api.Charger interface
func (c *OpenEVSE) Enabled() (bool, error) {
	res, err := c.statusG.Get()
	return *res.Status == openevse.Enabled, err
}

// Enable implements the api.Charger interface
func (c *OpenEVSE) Enable(enable bool) error {
	c.enabled = enable
	return c.setOverride()
}

// MaxCurrent implements the api.Charger interface
func (c *OpenEVSE) MaxCurrent(current int64) error {
	c.current = int(current)
	return c.setOverride()
}

var _ api.ChargeRater = (*OpenEVSE)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *OpenEVSE) ChargedEnergy() (float64, error) {
	res, err := c.statusG.Get()
	if err != nil {
		return 0, err
	}

	return *res.SessionEnergy / 1e3, err
}

var _ api.ChargeTimer = (*OpenEVSE)(nil)

func (c *OpenEVSE) ChargingTime() (time.Duration, error) {
	res, err := c.statusG.Get()
	if err != nil {
		return 0, err
	}
	return time.Duration(*res.SessionElapsed) * time.Second, err
}

var _ api.MeterEnergy = (*OpenEVSE)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *OpenEVSE) TotalEnergy() (float64, error) {
	res, err := c.statusG.Get()
	if err != nil {
		return 0, err
	}

	return *res.TotalEnergy, err
}

var _ api.Meter = (*OpenEVSE)(nil)

func (c *OpenEVSE) CurrentPower() (float64, error) {
	res, err := c.statusG.Get()
	if err != nil {
		return 0, err
	}
	return *res.Power, err
}

// phases1p3p implements the api.ChargePhases interface
func (c *OpenEVSE) phases1p3p(phases int) error {
	var set3p int
	if phases == 3 {
		set3p = 1
	}

	return c.rapiCommand(fmt.Sprintf("$S7 %d", set3p))
}

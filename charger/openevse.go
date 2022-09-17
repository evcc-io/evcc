package charger

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/deepmap/oapi-codegen/pkg/securityprovider"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openevse"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// OpenEVSE charger implementation
type OpenEVSE struct {
	uri     string
	api     *openevse.ClientWithResponses
	helper  *request.Helper
	timeout time.Duration
}

func init() {
	registry.Add("openevse", NewOpenEVSEFromConfig)
}

// go:generate go run ../cmd/tools/decorate.go -f decorateOpenEVSE -b ""*OpenEVSE" -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) (error)"

// NewOpenEVSEFromConfig creates a go-e charger from generic config
func NewOpenEVSEFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI      string
		User     string
		Password string
		Timeout  time.Duration
	}{
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return NewOpenEVSE(cc.URI, cc.User, cc.Password, cc.Timeout)
}

// NewOpenEVSE creates OpenEVSE charger
func NewOpenEVSE(uri, user, password string, timeout time.Duration) (api.Charger, error) {
	log := util.NewLogger("openevse").Redact(user, password)
	c := &OpenEVSE{
		helper:  request.NewHelper(log),
		uri:     uri,
		timeout: timeout,
	}

	options := []openevse.ClientOption{openevse.WithHTTPClient(c.helper.Client)}

	if user != "" && password != "" {
		basicAuthProvider, err := securityprovider.NewSecurityProviderBasicAuth(user, password)
		if err != nil {
			return c, err
		}

		options = append(options, openevse.WithRequestEditorFn(basicAuthProvider.Intercept))
	}

	var err error
	c.api, err = openevse.NewClientWithResponses(uri, options...)
	if err != nil {
		return c, err
	}

	var phases1p3p func(int) error

	ok, err := c.hasPhaseSwitchCapabilities()
	if ok {
		phases1p3p = c.phases1p3p

		// disable EVSE's own 1/3-phase auto-switching
		_, err := c.PerformRAPICommand(c.uri, "$S8 0")
		if err != nil {
			return c, err
		}
	}

	return decorateOpenEVSE(c, phases1p3p), err
}

func (c *OpenEVSE) requestContextWithTimeout() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	return ctx, cancel
}

func (c *OpenEVSE) hasPhaseSwitchCapabilities() (threePhaseCapabilities bool, err error) {
	_, err = c.PerformRAPICommand(c.uri, "$G7")
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c *OpenEVSE) PerformRAPICommand(uri, command string) (success bool, err error) {
	var uriBuilder strings.Builder
	uriBuilder.WriteString(uri)
	uriBuilder.WriteString("/r?json=1&rapi=")
	uriBuilder.WriteString(url.QueryEscape(command))

	var resp struct {
		Cmd, Ret string
	}

	err = c.helper.GetJSON(uriBuilder.String(), &resp)
	if err == nil && strings.HasPrefix(resp.Ret, "$OK") {
		return true, nil
	}

	return false, err
}

func (c *OpenEVSE) SetManualOverride(enable bool) error {
	state := "disabled"
	if enable {
		state = "active"
	}

	data := openevse.SetManualOverrideJSONRequestBody{
		State: &state,
	}

	ctx, cancel := c.requestContextWithTimeout()
	defer cancel()
	_, err := c.api.SetManualOverrideWithResponse(ctx, data)

	return err
}

func (c *OpenEVSE) Status() (api.ChargeStatus, error) {
	ctx, cancel := c.requestContextWithTimeout()
	defer cancel()
	resp, err := c.api.GetStatusWithResponse(ctx)
	if err != nil {
		return api.StatusNone, err
	}

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

	state := *resp.JSON200.State
	vehicleConnected := *resp.JSON200.Vehicle != 0

	switch state {
	case 1:
		return api.StatusA, nil
	case 2, 254, 255:
		if vehicleConnected {
			return api.StatusB, nil
		} else {
			return api.StatusA, nil
		}
	case 3:
		return api.StatusC, nil
	case 4:
		return api.StatusD, nil
	case 5, 6, 7, 8, 9, 10, 11:
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown EVSE state: %d", state)
	}
}

// Enabled implements the api.Charger interface
// Since OpenEVSE has many possible states, we manually override the state do it cannot change automatically
// For example, there can be an energy limit set, or a timer to only charge between certain hours
// Logic is: if it has manual override, keep it; otherwise enforce the current state as a manual override
func (c *OpenEVSE) Enabled() (bool, error) {
	ctx, cancel := c.requestContextWithTimeout()
	defer cancel()

	overrideResp, err := c.api.GetManualOverrideWithResponse(ctx)
	if err != nil {
		return false, err
	}

	if overrideResp.JSON200 != nil && overrideResp.JSON200.State != nil {
		switch *overrideResp.JSON200.State {
		case "disabled":
			return false, nil
		case "enabled", "active":
			return true, nil
		}
	}

	statusResp, err := c.api.GetStatusWithResponse(ctx)
	if err != nil {
		return false, err
	}

	var stateCode int
	if statusResp.JSON200 != nil && statusResp.JSON200.State != nil {
		stateCode = *statusResp.JSON200.State
	}

	var state bool
	switch stateCode {
	case 3, 4:
		state = true
	default:
		configResp, err := c.api.GetConfigWithResponse(ctx)
		if err != nil {
			return false, err
		}

		switch *configResp.JSON200.ChargeMode {
		case "fast":
			state = true
		}
	}

	err = c.SetManualOverride(state)

	return state, err
}

// Enable implements the api.Charger interface
func (c *OpenEVSE) Enable(enable bool) error {
	return c.SetManualOverride(enable)
}

// MaxCurrent implements the api.Charger interface
func (c *OpenEVSE) MaxCurrent(current int64) error {
	cur := int(current)
	data := openevse.SetManualOverrideJSONRequestBody{
		ChargeCurrent: &cur,
	}

	ctx, cancel := c.requestContextWithTimeout()
	defer cancel()

	_, err := c.api.SetManualOverrideWithResponse(ctx, data)

	return err
}

var _ api.ChargeRater = (*OpenEVSE)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *OpenEVSE) ChargedEnergy() (float64, error) {
	ctx, cancel := c.requestContextWithTimeout()
	defer cancel()

	resp, err := c.api.GetStatusWithResponse(ctx)
	if err != nil {
		return 0, err
	}

	return float64(*resp.JSON200.Wattsec) / 3600 / 1e3, nil
}

var _ api.MeterEnergy = (*OpenEVSE)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *OpenEVSE) TotalEnergy() (float64, error) {
	ctx, cancel := c.requestContextWithTimeout()
	defer cancel()

	resp, err := c.api.GetStatusWithResponse(ctx)
	if err != nil {
		return 0, err
	}

	return float64(*resp.JSON200.Watthour) / 1e3, nil
}

// phases1p3p implements the api.ChargePhases interface
func (c *OpenEVSE) phases1p3p(phases int) error {
	var set3p int
	if phases == 3 {
		set3p = 1
	}

	ok, err := c.PerformRAPICommand(c.uri, fmt.Sprintf("$S7 %d", set3p))
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("failed to switch to %d phase(s)", phases)
	}

	return nil
}

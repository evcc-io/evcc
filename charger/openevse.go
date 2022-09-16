package charger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/deepmap/oapi-codegen/pkg/securityprovider"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openevse"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// OpenEVSE charger implementation
type OpenEVSE struct {
	uri    string
	phases int
	api    *openevse.ClientWithResponses
	log    *util.Logger
	api.MeterEnergy
}

func init() {
	registry.Add("openevse", NewOpenEVSEFromConfig)
}

// go:generate go run oapi-codegen -package openevse -old-config-style -generate "types,client" openevse/api.yaml > openevse/api.go
// go:generate go run ../cmd/tools/decorate.go -f decorateOpenEVSE -b ""*OpenEVSE" -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) (error)"

// NewOpenEVSEFromConfig creates a go-e charger from generic config
func NewOpenEVSEFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI       string
		User      string
		Password  string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("must define uri")
	}

	return NewOpenEVSE(cc.URI, cc.User, cc.Password)
}

// NewOpenEVSE creates OpenEVSE charger
func NewOpenEVSE(uri, user, password string) (api.Charger, error) {
	c := &OpenEVSE{
		uri: uri,
		log: util.NewLogger("openevse").Redact(user, password),
	}

	var err error

	//conn.Logger(log.TRACE)

    loggingClientOption := openevse.WithHTTPClient(request.NewClient(c.log))

	if user != "" && password != "" {
		basicAuthProvider, err := securityprovider.NewSecurityProviderBasicAuth(user, password)

		if err != nil {
			return c, err
		}

		authOption := openevse.WithRequestEditorFn(basicAuthProvider.Intercept)
		c.api, err = openevse.NewClientWithResponses(uri, loggingClientOption, authOption)

		if err != nil {
			return c, err
		}
	} else {
		c.api, err = openevse.NewClientWithResponses(uri, loggingClientOption)

		if err != nil {
			return c, err
		}
	}

	var phaseSwitchFn func(int) error

	c.phases, err = c.DetectCapabilities()

	if err != nil {
		return c, err
	}

	if c.phases == 0 {
		phaseSwitchFn = c.phases1p3p
	}

	return decorateOpenEVSE(c, phaseSwitchFn), nil
}

func (c *OpenEVSE) DetectCapabilities() (phases int, err error) {
	_, err = c.IsChargingOnThreePhases()
	if err == nil {
		// phase switch supported
		return 0, nil
	}

	configResp, err := c.api.GetConfigWithResponse(context.Background())
	if err != nil {
		return 1, err
	}

	firmware := string(*configResp.JSON200.Firmware)
	regex := regexp.MustCompile(`\.3P`)
	matches := regex.FindStringSubmatch(firmware)

	if len(matches) != 0 {
		// 3-phase supported, assume actual 3-phase connection
		return 3, nil
	} else {
		return 1, nil
	}
}

func (c *OpenEVSE) IsChargingOnThreePhases() (threePhase bool, err error) {
	threePhaseResponse, _, err := c.PerformRAPICommand(c.uri, "$G7")
	if err != nil {
		return false, err
	}

	threePhaseInt, err := strconv.Atoi(threePhaseResponse)
	if err != nil {
		return false, err
	}

	return threePhaseInt != 0, nil
}

func (c *OpenEVSE) Phases() (phases int, err error) {
	phases = c.phases
	if phases != 0 {
		return phases, nil
	}

	isChargingOnThreePhase, err := c.IsChargingOnThreePhases()
	if err != nil {
		return 0, err
	}

	if isChargingOnThreePhase {
		phases = 3
	} else {
		phases = 1
	}

	return phases, nil
}

func (c *OpenEVSE) PerformRAPICommand(uri, command string) (response string, success bool, err error) {
	var uriBuilder strings.Builder
	uriBuilder.WriteString(uri)
	uriBuilder.WriteString("/r?json=1&rapi=")
	uriBuilder.WriteString(url.QueryEscape(command))
	rsp, err := http.DefaultClient.Get(uriBuilder.String())

	bodyBytes, err := ioutil.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return "", false, err
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var responseJson struct {
			Cmd *string
			Ret *string
		}
		if err := json.Unmarshal(bodyBytes, &responseJson); err != nil {
			return "", false, err
		}

		regex := regexp.MustCompile(`\$(OK|NK)([^^]*)\^\d+`)

		if regex == nil {
			return "", false, fmt.Errorf("invalid response from RAPI command %s: %s", command, string(bodyBytes))
		}

		matches := regex.FindStringSubmatch(*responseJson.Ret)

		if len(matches) == 0 {
			return "", false, fmt.Errorf("invalid response from RAPI command %s: %s", command, string(bodyBytes))
		}

		if matches[1] == "OK" {
			return strings.TrimSpace(matches[2]), true, nil
		} else {
			return "", false, nil
		}
	}

	return "", false, fmt.Errorf("invalid response from RAPI command %s: %s, code %d", command, rsp.Header.Get("Content-Type"), rsp.StatusCode)
}

func (c *OpenEVSE) SetManualOverride(enable bool) error {
	var state openevse.SetManualOverrideJSONBodyState
	if enable {
		state = "active"
	} else {
		state = "disabled"
	}

	body := openevse.SetManualOverrideJSONRequestBody{
		State: &state,
	}

	_, err := c.api.SetManualOverrideWithResponse(context.Background(), body)

	return err
}

func (c *OpenEVSE) Status() (api.ChargeStatus, error) {
	resp, err := c.api.GetStatusWithResponse(context.Background())
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
func (c *OpenEVSE) Enabled() (bool, error) {
	overrideResp, err := c.api.GetManualOverrideWithResponse(context.Background())

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

	statusResp, err := c.api.GetStatusWithResponse(context.Background())
	if err != nil {
		return false, err
	}

	var stateCode int
	if statusResp.JSON200 != nil && statusResp.JSON200.State != nil {
		stateCode = *statusResp.JSON200.State
	} else {
		stateCode = -1
	}

	var state bool
	switch stateCode {
	case 3, 4:
		state = true
	default:
		configResp, err := c.api.GetConfigWithResponse(context.Background())

		if err != nil {
			return false, err
		}

		switch *configResp.JSON200.ChargeMode {
			case "fast":
				state = true
			default:
				state = false
		}
	}

	err = c.SetManualOverride(state)

	return state, err
}

// Enable implements the api.Charger interface
func (c *OpenEVSE) Enable(enable bool) error {
	if enable && c.phases == 0 {
		_, success, err := c.PerformRAPICommand(c.uri, "$S8 0")

		if err != nil {
			return err
		}

		if !success {
			return fmt.Errorf("failed to turn off three phase auto-switching")
		}
	}

	err := c.SetManualOverride(enable)

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *OpenEVSE) MaxCurrent(current int64) error {
	cur := int(current)
	body := openevse.SetManualOverrideJSONRequestBody{
		ChargeCurrent: &cur,
	}

	_, err := c.api.SetManualOverrideWithResponse(context.Background(), body)

	return err
}

var _ api.Meter = (*OpenEVSE)(nil)

// CurrentPower implements the api.Meter interface
func (c *OpenEVSE) CurrentPower() (power float64, err error) {
	resp, err := c.api.GetStatusWithResponse(context.Background())
	if err != nil {
		return 0, err
	}

	phases, err := c.Phases()
	if err != nil {
		return 0, err
	}

	current := float64(phases) * float64(*resp.JSON200.Voltage) * float64(*resp.JSON200.Amp) / 1000

	return current, nil
}

var _ api.ChargeRater = (*OpenEVSE)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *OpenEVSE) ChargedEnergy() (float64, error) {
	resp, err := c.api.GetStatusWithResponse(context.Background())
	if err != nil {
		return 0, err
	}

	return float64(*resp.JSON200.Wattsec) / 3600 / 1000, nil
}

var _ api.MeterCurrent = (*OpenEVSE)(nil)

// Currents implements the api.MeterCurrent interface
func (c *OpenEVSE) Currents() (float64, float64, float64, error) {
	resp, err := c.api.GetStatusWithResponse(context.Background())
	if err != nil {
		return 0, 0, 0, err
	}

	cur := float64(*resp.JSON200.Amp) / 1000

	phases, err := c.Phases()
	if err != nil {
		return 0, 0, 0, err
	}

	if phases == 3 {
		return cur, cur, cur, nil
	} else {
		return cur, 0, 0, nil
	}
}

var _ api.Identifier = (*OpenEVSE)(nil)

// Identify implements the api.Identifier interface
func (c *OpenEVSE) Identify() (string, error) {
	return "", nil
}

// TotalEnergy implements the api.MeterEnergy interface
func (c *OpenEVSE) TotalEnergy() (float64, error) {
	resp, err := c.api.GetStatusWithResponse(context.Background())
	if err != nil {
		return 0, err
	}

	return float64(*resp.JSON200.Watthour) / 1000, nil
}

// phases1p3p implements the api.ChargePhases interface
func (c *OpenEVSE) phases1p3p(phases int) error {
	var enableThreePhases int
	if phases == 3 {
		enableThreePhases = 1
	} else {
		enableThreePhases = 0
	}

	_, success, err := c.PerformRAPICommand(c.uri, fmt.Sprintf("$S7 %d", enableThreePhases))

	if err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("failed to switch to %d phase(s)", phases)
	}

	return nil
}

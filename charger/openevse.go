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
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// OpenEVSE charger implementation
type OpenEVSE struct {
	uri                       string
	phases                    int
	autoPhasesSwitchSupported bool
	api                       *openevse.ClientWithResponses
	api.MeterEnergy
}

func init() {
	registry.Add("openevse", NewOpenEVSEFromConfig)
}

// go:generate go run ../cmd/tools/decorate.go -f decorateOpenEVSE -b "*OpenEVSE" -r api.Charger -t "api.ChargePhases,Phases1p3p,func(int) (error)"
// go:generate go run oapi-codegen -package openevse -old-config-style -generate "types,client" openevse/api.yaml > openevse/api.go

// NewOpenEVSEFromConfig creates a go-e charger from generic config
func NewOpenEVSEFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI            string
		User, Password string
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
	}
	var err error

	//log := util.NewLogger("openevse").Redact(password)

	basicAuthProvider, err := securityprovider.NewSecurityProviderBasicAuth(user, password)

	if err != nil {
		return c, err
	}

	if uri != "" {
		c.api, err = openevse.NewClientWithResponses(uri, openevse.WithRequestEditorFn(basicAuthProvider.Intercept))
	} else {
		c.api, err = openevse.NewClientWithResponses(uri)
	}

	var phaseSwitchFn func(int) error

	threePhaseSupport, _, err := c.DetectCapabilities(uri)
	if err == nil {
		if threePhaseSupport {
			c.phases = 3
		} else {
			c.phases = 1
		}

		phaseSwitchFn = c.phases1p3p
		c.autoPhasesSwitchSupported = true

		fmt.Println(c)
	} else {
		c.autoPhasesSwitchSupported = false
	}

	return decorateOpenEVSE(c, phaseSwitchFn), nil
}

func (c *OpenEVSE) DetectCapabilities(uri string) (threePhaseSupport, threePhaseAutoSwitch bool, err error) {
	isChargingOnThreePhase, err := c.IsChargingOnThreePhases(uri)
	if err != nil {
		return false, false, err
	}

	threePhaseAutoSwitch, err = c.HasThreePhaseAutoSwitch(uri)
	if err != nil {
		return false, false, err
	}

	//resp, err := c.api.GetStatusWithResponse(context.Background(), nil)
	//if err != nil {
	//	return false, false, err
	//}

	if threePhaseAutoSwitch || isChargingOnThreePhase {
		threePhaseSupport = true
	}

	return threePhaseSupport, threePhaseAutoSwitch, nil
}

func (c *OpenEVSE) IsChargingOnThreePhases(uri string) (threePhase bool, err error) {
	threePhaseResponse, _, err := c.PerformRAPICommand(uri, "$G7")
	if err != nil {
		return false, err
	}

	threePhaseInt, err := strconv.Atoi(threePhaseResponse)
	if err != nil {
		return false, err
	}

	return threePhaseInt != 0, nil
}

func (c *OpenEVSE) HasThreePhaseAutoSwitch(uri string) (threePhaseAutoSwitch bool, err error) {
	threePhaseAutoSwitchResponse, _, err := c.PerformRAPICommand(uri, "$G8")
	if err != nil {
		return false, err
	}

	threePhaseAutoSwitchInt, err := strconv.Atoi(threePhaseAutoSwitchResponse)
	if err != nil {
		return false, err
	}

	return threePhaseAutoSwitchInt != 0, nil
}

func (c *OpenEVSE) PerformRAPICommand(uri, command string) (response string, success bool, err error) {
	var uriBuilder strings.Builder
	uriBuilder.WriteString(uri)
	uriBuilder.WriteString("/r?json=1&rapi=")
	uriBuilder.WriteString(command)
	//req, err := http.NewRequest("GET", uriBuilder.String(), nil)
	//if err != nil {
	//	return false, false, err
	//}
	//
	//if err != nil {
	//	return false, false, err
	//}

	rsp, err := http.DefaultClient.Get(uriBuilder.String())

	fmt.Println(uriBuilder.String())

	bodyBytes, err := ioutil.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return "", false, err
	}

	fmt.Println(string(bodyBytes))

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var responseJson struct {
			Cmd *string
			Ret *string
		}
		if err := json.Unmarshal(bodyBytes, &responseJson); err != nil {
			return "", false, err
		}

		regex := regexp.MustCompile("\\$(OK|NK)\\s([^^]*)\\^\\d+")

		if regex == nil {
			return "", false, fmt.Errorf("invalid response from RAPI command %s: %s", command, string(bodyBytes))
		}

		matches := regex.FindStringSubmatch(*responseJson.Ret)

		if len(matches) == 0 {
			return "", false, fmt.Errorf("invalid response from RAPI command %s: %s", command, string(bodyBytes))
		}

		if matches[1] == "OK" {
			return matches[2], true, nil
		} else {
			return "", false, nil
		}
	}

	return "", false, fmt.Errorf("invalid response from RAPI command %s: %s, code %d", command, rsp.Header.Get("Content-Type"), rsp.StatusCode)
}

// Status implements the api.Charger interface
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

	switch car := *resp.JSON200.State; car {

	case 1:
		return api.StatusA, nil
	case 2, 254, 255:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	case 4:
		return api.StatusD, nil
	//case 255:
	//	return api.StatusE, nil
	case 5, 6, 7, 8, 9, 10, 11:
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("car unknown result: %d", car)
	}
}

// Enabled implements the api.Charger interface
func (c *OpenEVSE) Enabled() (bool, error) {
	overrideResp, err := c.api.GetManualOverrideWithResponse(context.Background())

	if err != nil {
		return false, err
	}

	fmt.Println(string(overrideResp.Body))

	if overrideResp.JSON200 != nil && overrideResp.JSON200.State != nil {
		switch *overrideResp.JSON200.State {
		case "disabled":
			return false, nil
		case "enabled", "active":
			return true, nil
		}
	}

	configResp, err := c.api.GetConfigWithResponse(context.Background())
	if err != nil {
		return false, err
	}

	//fmt.Println(string(configResp.Body))

	switch *configResp.JSON200.ChargeMode {
	case "fast":
		return true, nil
	default:
		return false, nil
	}
}

// Enable implements the api.Charger interface
func (c *OpenEVSE) Enable(enable bool) error {
	overrideResp, err := c.api.GetManualOverrideWithResponse(context.Background())

	if err != nil {
		return err
	}

	fmt.Println(string(overrideResp.Body))

	if overrideResp.JSON200 != nil && overrideResp.JSON200.State != nil {
		clearOverrideResp, err := c.api.ClearManualOverrideWithResponse(context.Background())
		if err != nil {
			return err
		}

		if clearOverrideResp.StatusCode() != 200 {
			if overrideResp.JSON200 != nil && overrideResp.JSON200.State != nil {
				return fmt.Errorf("cannot clear %s override", overrideResp.JSON200.State)
			} else {
				return fmt.Errorf("cannot clear override")
			}
		}
	}

	if enable && c.autoPhasesSwitchSupported {
		_, success, err := c.PerformRAPICommand(c.uri, fmt.Sprintf("$S8 0"))

		if err != nil {
			return err
		}

		if success != true {
			return fmt.Errorf("failed to turn off three phase auto-switching")
		}
	}

	enabled, err := c.Enabled()
	if enabled == enable && err == nil {
		// already at the desired state
		// don't throw exception yet, just fallback to a manual override
		return nil
	}

	var state openevse.SetManualOverrideJSONBodyState
	if enable {
		state = "active"
	} else {
		state = "disabled"
	}

	body := openevse.SetManualOverrideJSONRequestBody{
		State: &state,
	}

	fmt.Println(body)
	resp, err := c.api.SetManualOverrideWithResponse(context.Background(), body)
	fmt.Println(resp.Body)

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
func (c *OpenEVSE) CurrentPower() (float64, error) {
	resp, err := c.api.GetStatusWithResponse(context.Background())
	if err != nil {
		return 0, err
	}

	return float64(c.phases) * float64(*resp.JSON200.Voltage) * float64(*resp.JSON200.Amp) / 1000, nil
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

	if c.phases == 3 {
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

// phases1p3p implements the api.ChargePhases interface - v2 only
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

	if success != true {
		return fmt.Errorf("failed to switch to %d phase(s)", phases)
	}

	return nil
}

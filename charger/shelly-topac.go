package charger

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

// ShellyTopAC charger implementation for Shelly Top AC Portable EV Charger
// API Reference: https://shelly-api-docs.shelly.cloud/gen2/Devices/ShellyX/XT1/TopACPortableEVCharger/
type ShellyTopAC struct {
	*request.Helper
	uri     string
	current int64
	statusG util.Cacheable[topACStatus]
	phaseG  util.Cacheable[topACPhaseInfo]
	energyG util.Cacheable[topACEnergy]
}

type topACRpcRequest struct {
	Id     int    `json:"id"`
	Src    string `json:"src"`
	Method string `json:"method"`
	Params struct {
		Owner string `json:"owner"`
		Role  string `json:"role"`
		Value any    `json:"value,omitempty"`
	} `json:"params"`
}

type topACEnumResponse struct {
	Value        string `json:"value"`
	Source       string `json:"source"`
	LastUpdateTs int64  `json:"last_update_ts"`
}

type topACNumberResponse struct {
	Value        float64 `json:"value"`
	Source       string  `json:"source"`
	LastUpdateTs int64   `json:"last_update_ts"`
}

type topACPhaseData struct {
	Voltage float64 `json:"voltage"`
	Current float64 `json:"current"`
	Power   float64 `json:"power"`
}

type topACPhaseInfoValue struct {
	TotalCurrent   float64        `json:"total_current"`
	TotalPower     float64        `json:"total_power"`
	TotalActEnergy float64        `json:"total_act_energy"`
	PhaseA         topACPhaseData `json:"phase_a"`
	PhaseB         topACPhaseData `json:"phase_b"`
	PhaseC         topACPhaseData `json:"phase_c"`
}

type topACObjectResponse struct {
	Value        topACPhaseInfoValue `json:"value"`
	Source       string              `json:"source"`
	LastUpdateTs int64               `json:"last_update_ts"`
}

type topACStatus struct {
	workState string
}

type topACPhaseInfo struct {
	info topACPhaseInfoValue
}

type topACEnergy struct {
	sessionEnergy float64
}

type topACServiceConfigRequest struct {
	Id     int                   `json:"id"`
	Src    string                `json:"src"`
	Method string                `json:"method"`
	Params topACServiceConfigSet `json:"params"`
}

type topACServiceConfigSet struct {
	Id         int  `json:"id"`
	AutoCharge bool `json:"auto_charge"`
}

func init() {
	registry.Add("shelly-topac", NewShellyTopACFromConfig)
}

// NewShellyTopACFromConfig creates a Shelly Top AC charger from generic config
func NewShellyTopACFromConfig(other map[string]any) (api.Charger, error) {
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

	return NewShellyTopAC(cc.URI, cc.User, cc.Password, cc.Cache)
}

// NewShellyTopAC creates Shelly Top AC charger
func NewShellyTopAC(uri, user, password string, cache time.Duration) (api.Charger, error) {
	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	// normalize URI
	for _, suffix := range []string{"/", "/rpc", "/shelly"} {
		uri = strings.TrimSuffix(uri, suffix)
	}
	uri = util.DefaultScheme(uri, "http")

	log := util.NewLogger("topac")
	helper := request.NewHelper(log)
	helper.Transport = request.NewTripper(log, transport.Insecure())

	c := &ShellyTopAC{
		Helper:  helper,
		uri:     fmt.Sprintf("%s/rpc", uri),
		current: 6, // default minimum current
	}

	// Setup digest authentication for Shelly Gen2
	if user != "" {
		c.Client.Transport = digest.NewTransport(user, password, c.Client.Transport)
	}

	// Setup cached status getters
	c.statusG = util.ResettableCached(c.getWorkState, cache)
	c.phaseG = util.ResettableCached(c.getPhaseInfo, cache)
	c.energyG = util.ResettableCached(c.getSessionEnergy, cache)

	// Enable auto_charge configuration
	if err := c.setAutoCharge(true); err != nil {
		log.WARN.Printf("failed to enable auto_charge: %v", err)
	}

	return c, nil
}

// execRpc executes a Shelly Gen2 RPC call
func (c *ShellyTopAC) execRpc(method, owner, role string, value any, res any) error {
	data := topACRpcRequest{
		Id:     0,
		Src:    "evcc",
		Method: method,
	}
	data.Params.Owner = owner
	data.Params.Role = role
	data.Params.Value = value

	req, err := request.New(http.MethodPost, fmt.Sprintf("%s/%s", c.uri, method), request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	return c.DoJSON(req, &res)
}

// getWorkState retrieves the charger's work state
func (c *ShellyTopAC) getWorkState() (topACStatus, error) {
	var res topACEnumResponse
	err := c.execRpc("Enum.GetStatus", "service:0", "work_state", nil, &res)

	return topACStatus{workState: res.Value}, err
}

// getCurrentLimit retrieves the current charging limit
func (c *ShellyTopAC) getCurrentLimit() (float64, error) {
	var res topACNumberResponse
	err := c.execRpc("Number.GetStatus", "service:0", "current_limit", nil, &res)

	return res.Value, err
}

// setCurrentLimit sets the charging current limit
func (c *ShellyTopAC) setCurrentLimit(current float64) error {
	var res any

	return c.execRpc("Number.Set", "service:0", "current_limit", current, &res)
}

// getPhaseInfo retrieves phase information
func (c *ShellyTopAC) getPhaseInfo() (topACPhaseInfo, error) {
	var res topACObjectResponse
	err := c.execRpc("Object.GetStatus", "service:0", "phase_info", nil, &res)

	return topACPhaseInfo{info: res.Value}, err
}

// getSessionEnergy retrieves the session energy consumption
func (c *ShellyTopAC) getSessionEnergy() (topACEnergy, error) {
	var res topACNumberResponse
	err := c.execRpc("Number.GetStatus", "service:0", "energy_charge", nil, &res)

	return topACEnergy{sessionEnergy: res.Value}, err
}

// setAutoCharge enables or disables auto charge configuration
func (c *ShellyTopAC) setAutoCharge(enable bool) error {
	data := topACServiceConfigRequest{
		Id:     0,
		Src:    "evcc",
		Method: "Service.SetConfig",
		Params: topACServiceConfigSet{
			Id:         0,
			AutoCharge: enable,
		},
	}

	req, err := request.New(http.MethodPost, fmt.Sprintf("%s/Service.SetConfig", c.uri), request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	var res any
	return c.DoJSON(req, &res)
}

// Status implements the api.Charger interface
func (c *ShellyTopAC) Status() (api.ChargeStatus, error) {
	status, err := c.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	// Map Shelly work states to evcc charge status
	// Possible states: charger_free, charger_wait, charger_pause, charger_charging, charger_complete, charger_error
	switch status.workState {
	case "charger_free":
		return api.StatusA, nil
	case "charger_wait", "charger_pause", "charger_complete":
		return api.StatusB, nil
	case "charger_charging":
		return api.StatusC, nil
	case "charger_error":
		return api.StatusE, fmt.Errorf("charger error")
	default:
		return api.StatusNone, fmt.Errorf("unknown work state: %s", status.workState)
	}
}

// Enabled implements the api.Charger interface
func (c *ShellyTopAC) Enabled() (bool, error) {
	current, err := c.getCurrentLimit()
	if err != nil {
		return false, err
	}

	// Charger is enabled if current limit > 0
	return current > 0, nil
}

// Enable implements the api.Charger interface
func (c *ShellyTopAC) Enable(enable bool) error {
	var current int64
	if enable {
		current = c.current
	}

	c.statusG.Reset()

	return c.setCurrentLimit(float64(current))
}

// MaxCurrent implements the api.Charger interface
func (c *ShellyTopAC) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	c.current = current

	// Only set if charger is enabled
	enabled, err := c.Enabled()
	if err != nil {
		return err
	}

	if enabled {
		return c.setCurrentLimit(float64(current))
	}

	return nil
}

var _ api.Meter = (*ShellyTopAC)(nil)

// CurrentPower implements the api.Meter interface
func (c *ShellyTopAC) CurrentPower() (float64, error) {
	phase, err := c.phaseG.Get()
	if err != nil {
		return 0, err
	}

	return phase.info.TotalPower, nil
}

var _ api.MeterEnergy = (*ShellyTopAC)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *ShellyTopAC) TotalEnergy() (float64, error) {
	phase, err := c.phaseG.Get()
	if err != nil {
		return 0, err
	}

	return phase.info.TotalActEnergy, nil
}

var _ api.PhaseCurrents = (*ShellyTopAC)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *ShellyTopAC) Currents() (float64, float64, float64, error) {
	phase, err := c.phaseG.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	return phase.info.PhaseA.Current, phase.info.PhaseB.Current, phase.info.PhaseC.Current, nil
}

var _ api.PhaseVoltages = (*ShellyTopAC)(nil)

// Voltages implements the api.PhaseVoltages interface
func (c *ShellyTopAC) Voltages() (float64, float64, float64, error) {
	phase, err := c.phaseG.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	return phase.info.PhaseA.Voltage, phase.info.PhaseB.Voltage, phase.info.PhaseC.Voltage, nil
}

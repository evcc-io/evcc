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
	"github.com/evcc-io/evcc/charger/shelly"
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
	phaseG  util.Cacheable[shelly.Measurements]
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
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewShellyTopAC(cc.URI, cc.User, cc.Password)
}

// NewShellyTopAC creates Shelly Top AC charger
func NewShellyTopAC(uri, user, password string) (api.Charger, error) {
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
	c.phaseG = util.ResettableCached(c.getPhaseInfo, time.Second)

	return c, nil
}

// execRpc executes a Shelly Gen2 RPC call
func (c *ShellyTopAC) execRpc(method string, params, res any) error {
	data := shelly.RpcRequest{
		Id:     0,
		Src:    "evcc",
		Method: method,
		Params: params,
	}

	req, err := request.New(http.MethodPost, c.uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	return c.DoJSON(req, &res)
}

// setCurrentLimit sets the charging current limit
func (c *ShellyTopAC) setCurrentLimit(current float64) error {
	var res any
	params := shelly.SetValueParams{Owner: "service:0", Role: "current_limit", Value: current}

	return c.execRpc("Number.Set", params, &res)
}

// getPhaseInfo retrieves phase information
func (c *ShellyTopAC) getPhaseInfo() (shelly.Measurements, error) {
	var res shelly.RpcResponseWrapper[shelly.Measurements]
	params := shelly.RoleParams{Owner: "service:0", Role: "phase_info"}
	err := c.execRpc("Object.GetStatus", params, &res)

	return res.Result.Value, err
}

// Status implements the api.Charger interface
func (c *ShellyTopAC) Status() (api.ChargeStatus, error) {
	var res shelly.RpcResponseWrapper[string]
	params := shelly.RoleParams{Owner: "service:0", Role: "work_state"}
	if err := c.execRpc("Enum.GetStatus", params, &res); err != nil {
		return api.StatusNone, err
	}

	// Possible states: charger_free, charger_wait, charger_pause, charger_charging, charger_complete, charger_error
	switch res.Result.Value {
	case "charger_free":
		return api.StatusA, nil
	case "charger_wait", "charger_pause", "charger_complete":
		return api.StatusB, nil
	case "charger_charging":
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown work state: %s", res.Result.Value)
	}
}

// Enabled implements the api.Charger interface
func (c *ShellyTopAC) Enabled() (bool, error) {
	var res shelly.RpcResponseWrapper[float64]
	params := shelly.RoleParams{Owner: "service:0", Role: "current_limit"}
	err := c.execRpc("Number.GetStatus", params, &res)

	return res.Result.Value > 0, err
}

// Enable implements the api.Charger interface
func (c *ShellyTopAC) Enable(enable bool) error {
	var current int64
	if enable {
		current = c.current
	}

	return c.setCurrentLimit(float64(current))
}

// MaxCurrent implements the api.Charger interface
func (c *ShellyTopAC) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	err := c.setCurrentLimit(float64(current))
	if err == nil {
		c.current = current
	}

	return err
}

var _ api.Meter = (*ShellyTopAC)(nil)

// CurrentPower implements the api.Meter interface
func (c *ShellyTopAC) CurrentPower() (float64, error) {
	phase, err := c.phaseG.Get()
	if err != nil {
		return 0, err
	}

	return phase.TotalPower * 1e3, nil
}

var _ api.MeterEnergy = (*ShellyTopAC)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *ShellyTopAC) TotalEnergy() (float64, error) {
	res, err := c.phaseG.Get()
	if err != nil {
		return 0, err
	}

	return res.TotalActEnergy, nil
}

var _ api.PhaseCurrents = (*ShellyTopAC)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *ShellyTopAC) Currents() (float64, float64, float64, error) {
	res, err := c.phaseG.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	return res.PhaseA.Current, res.PhaseB.Current, res.PhaseC.Current, nil
}

var _ api.PhaseVoltages = (*ShellyTopAC)(nil)

// Voltages implements the api.PhaseVoltages interface
func (c *ShellyTopAC) Voltages() (float64, float64, float64, error) {
	res, err := c.phaseG.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	return res.PhaseA.Voltage, res.PhaseB.Voltage, res.PhaseC.Voltage, nil
}

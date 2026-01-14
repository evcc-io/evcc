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
	uri    string
	phaseG util.Cacheable[shelly.Measurements]
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
		Helper: helper,
		uri:    fmt.Sprintf("%s/rpc", uri),
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
func (c *ShellyTopAC) execRpc(method, owner, role string, value, res any) error {
	data := shelly.RpcRequest{
		Id:     0,
		Src:    "evcc",
		Method: method,
		Params: shelly.RpcRequestParams{
			Owner: owner,
			Role:  role,
			Value: value,
		},
	}

	req, err := request.New(http.MethodPost, c.uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	return c.DoJSON(req, res)
}

// getPhaseInfo retrieves phase information
func (c *ShellyTopAC) getPhaseInfo() (shelly.Measurements, error) {
	var res shelly.RpcResponse[shelly.Measurements]
	err := c.execRpc("Object.GetStatus", "service:0", "phase_info", nil, &res)

	return res.Result.Value, err
}

// Status implements the api.Charger interface
func (c *ShellyTopAC) Status() (api.ChargeStatus, error) {
	var res shelly.RpcResponse[string]
	if err := c.execRpc("Enum.GetStatus", "service:0", "work_state", nil, &res); err != nil {
		return api.StatusNone, err
	}

	// Possible states: charger_free, charger_wait, charger_pause, charger_charging, charger_complete, charger_error
	switch res.Result.Value {
	case "charger_free":
		return api.StatusA, nil
	case "charger_wait", "charger_pause", "charger_complete", "charger_end":
		return api.StatusB, nil
	case "charger_charging":
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown work state: %s", res.Result.Value)
	}
}

// Enabled implements the api.Charger interface
func (c *ShellyTopAC) Enabled() (bool, error) {
	var res shelly.RpcResponse[bool]
	err := c.execRpc("Boolean.GetStatus", "service:0", "start_charging", nil, &res)

	return res.Result.Value, err
}

// Enable implements the api.Charger interface
func (c *ShellyTopAC) Enable(enable bool) error {
	var res any
	return c.execRpc("Boolean.Set", "service:0", "start_charging", enable, &res)
}

// MaxCurrent implements the api.Charger interface
func (c *ShellyTopAC) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*ShellyTopAC)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *ShellyTopAC) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	var res any
	return c.execRpc("Number.Set", "service:0", "current_limit", current, &res)
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

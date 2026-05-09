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

// https://github.com/Lektrico/lektricowifi
//
// Lektrico charger protocol (discovered from lektricowifi source + real device testing):
//
//   GET  http://<host>/rpc/<Method>   -> direct JSON response
//   POST http://<host>/rpc            -> JSON-RPC body, response wrapped in "result" field
//
// Main endpoint: charger_info.get returns all data in a single request.
//
// Example response from charger_info.get:
//   {
//     "charger_state": "B",             // raw IEC state: A/B/C/D/E/F/B_AUTH/B_PAUSE/OTA/LOCKED
//     "extended_charger_state": "B_AUTH",
//     "session_energy": 38.48,          // Wh
//     "instant_power": 0.0,             // W
//     "currents": [0.0, 0.0, 0.0],      // A, array [L1, L2, L3]
//     "voltages": [237.65, 0.0, 0.0],   // V, array [L1, L2, L3]
//     "total_charged_energy": 9683.844, // kWh
//     "dynamic_current": 32,            // allowed current (0=pause, 6-32=active)
//     "relay_mode": 0,                  // phase mode
//     "has_active_errors": false,
//     "charger_is_paused": false,
//     "current_limit_reason": 2,        // int: 0=no_limit, 1=installation_current, 2=user_limit,
//                                       //      3=dynamic_limit, 4=schedule, 5=em_offline, 6=em,
//                                       //      7=ocpp, 8=overtemperature, 9=switching_phases,
//                                       //      10=user_limit, 11=1p_charging_disabled, 12+=unknown
//     "temperature": 18.8,
//     "fw_version": "1.51",
//     "headless": true,                 // true = no authentication required
//     "install_current": 32,
//   }

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

// lektricoStateBAUTH is the extended state indicating waiting for RFID authentication
const lektricoStateBAUTH = "B_AUTH"

// lektricoInfo maps the JSON response from charger_info.get
type lektricoInfo struct {
	ChargerState         string    `json:"charger_state"`
	ExtendedChargerState string    `json:"extended_charger_state"`
	HasActiveErrors      bool      `json:"has_active_errors"`
	InstantPower         float64   `json:"instant_power"`
	SessionEnergy        float64   `json:"session_energy"`
	TotalChargedEnergy   float64   `json:"total_charged_energy"`
	Currents             []float64 `json:"currents"`
	Voltages             []float64 `json:"voltages"`
	DynamicCurrent       int       `json:"dynamic_current"`
	FwVersion            string    `json:"fw_version"`
}

// lektricoRPCRequest is the POST JSON-RPC request format
type lektricoRPCRequest struct {
	Src    string         `json:"src"`
	ID     int            `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

// lektricoRPCResponse wraps the POST response
type lektricoRPCResponse struct {
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Lektrico implements api.Charger for Lektrico 1P7K / 3P22K charging stations
type Lektrico struct {
	*request.Helper
	rpcID   atomic.Uint32
	uri     string
	current int64
	statusG util.Cacheable[lektricoInfo]
}

var _ api.Charger = (*Lektrico)(nil)

func init() {
	registry.Add("lektrico", NewLektricoFromConfig)
}

// NewLektricoFromConfig creates a Lektrico charger from evcc configuration
func NewLektricoFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		Host  string
		Cache time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Host == "" {
		return nil, fmt.Errorf("missing host")
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	return NewLektrico(cc.Host, cc.Cache)
}

// NewLektrico creates a Lektrico charger and verifies connectivity
func NewLektrico(host string, cache time.Duration) (*Lektrico, error) {
	uri := fmt.Sprintf("http://%s/rpc", strings.TrimSuffix(host, "/"))

	wb := &Lektrico{
		Helper:  request.NewHelper(util.NewLogger("lektrico")),
		uri:     uri,
		current: 6,
	}

	wb.statusG = util.ResettableCached(func() (lektricoInfo, error) {
		var res lektricoInfo
		err := wb.GetJSON(uri+"/charger_info.get", &res)
		return res, err
	}, cache)

	return wb, nil
}

// post sends a JSON-RPC command to the charger
func (wb *Lektrico) post(method string, params map[string]any) error {
	payload := lektricoRPCRequest{
		Src:    "evcc",
		ID:     int(wb.rpcID.Add(1)),
		Method: method,
		Params: params,
	}

	req, _ := request.New(http.MethodPost, wb.uri, request.MarshalJSON(payload), request.JSONEncoding)

	var res lektricoRPCResponse
	if err := wb.DoJSON(req, &res); err != nil {
		return err
	}
	if res.Error != nil {
		return fmt.Errorf("%d: %s", res.Error.Code, res.Error.Message)
	}

	wb.statusG.Reset()
	return nil
}

// Status implements the api.Charger interface
func (wb *Lektrico) Status() (api.ChargeStatus, error) {
	res, err := wb.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	switch res.ChargerState {
	case "A":
		return api.StatusA, nil
	case "B":
		return api.StatusB, nil
	case "C":
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid state: %s", res.ChargerState)
	}
}

var _ api.StatusReasoner = (*Lektrico)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *Lektrico) StatusReason() (api.Reason, error) {
	res, err := wb.statusG.Get()
	if err == nil && res.ExtendedChargerState == lektricoStateBAUTH {
		return api.ReasonWaitingForAuthorization, nil
	}

	return api.ReasonUnknown, err
}

// Enabled implements the api.Charger interface
func (wb *Lektrico) Enabled() (bool, error) {
	res, err := wb.statusG.Get()
	if err != nil {
		return false, err
	}

	return res.DynamicCurrent > 0, nil
}

// sendCurrent sets the dynamic_current on the charger.
func (wb *Lektrico) setCurrent(value int64) error {
	return wb.post("dynamic_current.set", map[string]any{
		"dynamic_current": value,
	})
}

// Enable implements the api.Charger interface
func (wb *Lektrico) Enable(enable bool) error {
	var curr int64
	if enable {
		curr = wb.current
	}

	return wb.setCurrent(curr)
}

// MaxCurrent implements the api.Charger interface
func (wb *Lektrico) MaxCurrent(current int64) error {
	err := wb.setCurrent(current)
	if err == nil {
		wb.current = current
	}
	return err
}

var _ api.Meter = (*Lektrico)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Lektrico) CurrentPower() (float64, error) {
	res, err := wb.statusG.Get()
	return res.InstantPower, err
}

var _ api.MeterImport = (*Lektrico)(nil)

// ImportEnergy implements the api.MeterImport interface
func (wb *Lektrico) ImportEnergy() (float64, error) {
	res, err := wb.statusG.Get()
	return res.TotalChargedEnergy, err
}

var _ api.ChargeRater = (*Lektrico)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Lektrico) ChargedEnergy() (float64, error) {
	res, err := wb.statusG.Get()
	return res.SessionEnergy / 1000.0, err
}

var _ api.PhaseCurrents = (*Lektrico)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Lektrico) Currents() (float64, float64, float64, error) {
	res, err := wb.statusG.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	return res.Currents[0], res.Currents[1], res.Currents[2], nil
}

var _ api.PhaseVoltages = (*Lektrico)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Lektrico) Voltages() (float64, float64, float64, error) {
	res, err := wb.statusG.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	return res.Voltages[0], res.Voltages[1], res.Voltages[2], nil
}

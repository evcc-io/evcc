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

// lektricoRPCCounter provides monotonically increasing JSON-RPC request IDs
var lektricoRPCCounter atomic.Uint32

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
	log     *util.Logger
	uri     string
	current atomic.Int64
	phases  atomic.Int64
	statusG util.Cacheable[lektricoInfo]
}

var _ api.Charger = (*Lektrico)(nil)

func init() {
	registry.Add("lektrico", NewLektricoFromConfig)
}

// NewLektricoFromConfig creates a Lektrico charger from evcc configuration
func NewLektricoFromConfig(other map[string]any) (api.Charger, error) {
	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

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

	return NewLektrico(cc.Host, cc.Cache)
}

// NewLektrico creates a Lektrico charger and verifies connectivity
func NewLektrico(host string, cache time.Duration) (*Lektrico, error) {
	log := util.NewLogger("lektrico")
	uri := fmt.Sprintf("http://%s/rpc", strings.TrimSuffix(host, "/"))

	l := &Lektrico{
		Helper: request.NewHelper(log),
		log:    log,
		uri:    uri,
	}

	l.statusG = util.ResettableCached(func() (lektricoInfo, error) {
		var res lektricoInfo
		err := l.GetJSON(uri+"/charger_info.get", &res)
		return res, err
	}, cache)

	// Connectivity check
	var id struct {
		DeviceID string `json:"device_id"`
	}
	if err := l.GetJSON(uri+"/Device_id.Get", &id); err != nil {
		return nil, fmt.Errorf("lektrico: connection failed to %s: %w", host, err)
	}
	log.DEBUG.Printf("connected to Lektrico charger: %s", id.DeviceID)

	// Read initial dynamic_current
	if info, err := l.statusG.Get(); err == nil && info.DynamicCurrent > 0 {
		l.current.Store(int64(info.DynamicCurrent))
	}

	return l, nil
}

// post sends a JSON-RPC command to the charger
func (l *Lektrico) post(method string, params map[string]any) error {
	payload := lektricoRPCRequest{
		Src:    "evcc",
		ID:     int(lektricoRPCCounter.Add(1)),
		Method: method,
		Params: params,
	}

	req, err := request.New(http.MethodPost, l.uri, request.MarshalJSON(payload), request.JSONEncoding)
	if err != nil {
		return err
	}

	var resp lektricoRPCResponse
	if err := l.DoJSON(req, &resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("charger error: code=%d msg=%s", resp.Error.Code, resp.Error.Message)
	}

	l.statusG.Reset()
	return nil
}

// Status implements the api.Charger interface
func (l *Lektrico) Status() (api.ChargeStatus, error) {
	info, err := l.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	switch info.ChargerState {
	case "A":
		return api.StatusA, nil
	case "B":
		return api.StatusB, nil
	case "C":
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown charger state: %s", info.ChargerState)
	}
}

var _ api.StatusReasoner = (*Lektrico)(nil)

// StatusReason implements the api.StatusReasoner interface
func (l *Lektrico) StatusReason() (api.Reason, error) {
	info, err := l.statusG.Get()
	if err != nil {
		return api.ReasonUnknown, err
	}

	if info.ExtendedChargerState == lektricoStateBAUTH {
		return api.ReasonWaitingForAuthorization, nil
	}

	return api.ReasonUnknown, nil
}

// Enabled implements the api.Charger interface
func (l *Lektrico) Enabled() (bool, error) {
	info, err := l.statusG.Get()
	if err != nil {
		return false, err
	}

	return info.DynamicCurrent > 0, nil
}

// sendCurrent sets the dynamic_current on the charger.
// A value of 0 pauses charging; values in [lektricoMinCurrentA, lektricoMaxCurrentA] set the current.
func (l *Lektrico) sendCurrent(value int64) error {
	return l.post("dynamic_current.set", map[string]any{
		"dynamic_current": value,
	})
}

// Enable implements the api.Charger interface
func (l *Lektrico) Enable(enable bool) error {
	var curr int64
	if enable {
		curr = l.current.Load() // restore last current
	}

	return l.sendCurrent(curr)
}

// MaxCurrent implements the api.Charger interface
func (l *Lektrico) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	l.current.Store(current)
	return l.sendCurrent(current)
}

var _ api.PhaseSwitcher = (*Lektrico)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
// relay_mode values are unverified: 0=3-phase, 1=1-phase
func (l *Lektrico) Phases1p3p(phases int) error {
	relayMode := 0 // 3-phase
	if phases == 1 {
		relayMode = 1
	}
	if err := l.post("dynamic_current.set", map[string]any{
		"dynamic_current": l.current.Load(),
		"relay_mode":      relayMode,
	}); err != nil {
		return err
	}
	l.phases.Store(int64(phases))
	return nil
}

var _ api.Meter = (*Lektrico)(nil)

// CurrentPower implements the api.Meter interface
func (l *Lektrico) CurrentPower() (float64, error) {
	info, err := l.statusG.Get()
	return info.InstantPower, err
}

var _ api.MeterEnergy = (*Lektrico)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (l *Lektrico) TotalEnergy() (float64, error) {
	info, err := l.statusG.Get()
	return info.TotalChargedEnergy, err
}

var _ api.ChargeRater = (*Lektrico)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (l *Lektrico) ChargedEnergy() (float64, error) {
	info, err := l.statusG.Get()
	return info.SessionEnergy / 1000.0, err
}

var _ api.PhaseCurrents = (*Lektrico)(nil)

// Currents implements the api.PhaseCurrents interface
func (l *Lektrico) Currents() (float64, float64, float64, error) {
	info, err := l.statusG.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	if len(info.Currents) < 3 {
		return 0, 0, 0, fmt.Errorf("incomplete currents array (%d elements)", len(info.Currents))
	}
	return info.Currents[0], info.Currents[1], info.Currents[2], nil
}

var _ api.PhaseVoltages = (*Lektrico)(nil)

// Voltages implements the api.PhaseVoltages interface
func (l *Lektrico) Voltages() (float64, float64, float64, error) {
	info, err := l.statusG.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	if len(info.Voltages) < 3 {
		return 0, 0, 0, fmt.Errorf("incomplete voltages array (%d elements)", len(info.Voltages))
	}
	return info.Voltages[0], info.Voltages[1], info.Voltages[2], nil
}

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
//     "has_active_errors": false,
//     "charger_is_paused": false,
//     "current_limit_reason": 2,        // int (0=no_limit, 2=user_limit, ...)
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

// Current limits for Lektrico chargers
const (
	lektricoMinCurrentA int64 = 6
	lektricoMaxCurrentA int64 = 32
)

// lektricoRPCCounter provides monotonically increasing JSON-RPC request IDs
var lektricoRPCCounter atomic.Uint32

// Raw IEC states returned in extended_charger_state
const (
	lektricoIECA          = "A"
	lektricoIECB          = "B"
	lektricoIECBAUTH      = "B_AUTH"
	lektricoIECBPAUSE     = "B_PAUSE"
	lektricoIECBSCHEDULER = "B_SCHEDULER"
	lektricoIECC          = "C"
	lektricoIECD          = "D"
	lektricoIECE          = "E"
	lektricoIECF          = "F"
	lektricoIECOTA        = "OTA"
	lektricoIECLOCKED     = "LOCKED"
)

// lektricoInfo maps the JSON response from charger_info.get
type lektricoInfo struct {
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
	l.current.Store(lektricoMinCurrentA)

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

	if info.HasActiveErrors {
		return api.StatusE, nil
	}

	switch info.ExtendedChargerState {
	case lektricoIECA:
		return api.StatusA, nil
	case lektricoIECB, lektricoIECBAUTH, lektricoIECBPAUSE,
		lektricoIECBSCHEDULER, lektricoIECLOCKED:
		return api.StatusB, nil
	case lektricoIECC, lektricoIECD:
		return api.StatusC, nil
	case lektricoIECE, lektricoIECF:
		return api.StatusE, nil
	case lektricoIECOTA:
		return api.StatusB, nil
	default:
		l.log.WARN.Printf("unknown charger state: %s", info.ExtendedChargerState)
		return api.StatusNone, nil
	}
}

// Enabled implements the api.Charger interface
func (l *Lektrico) Enabled() (bool, error) {
	info, err := l.statusG.Get()
	if err != nil {
		return false, err
	}
	return info.DynamicCurrent >= int(lektricoMinCurrentA), nil
}

// sendCurrent sends dynamic_current and user_current to the charger.
// A value of 0 pauses charging; values in [lektricoMinCurrentA, lektricoMaxCurrentA] set the current.
func (l *Lektrico) sendCurrent(value int64) error {
	if err := l.post("dynamic_current.set", map[string]any{
		"dynamic_current": value,
	}); err != nil {
		return err
	}
	return l.post("app_config.set", map[string]any{
		"config_key":   "user_current",
		"config_value": value,
	})
}

// Enable implements the api.Charger interface
func (l *Lektrico) Enable(enable bool) error {
	var value int64
	if enable {
		value = l.current.Load()
		if value < lektricoMinCurrentA {
			value = lektricoMinCurrentA
		}
	}
	return l.sendCurrent(value)
}

// MaxCurrent implements the api.Charger interface
func (l *Lektrico) MaxCurrent(current int64) error {
	if current < lektricoMinCurrentA {
		current = lektricoMinCurrentA
	}
	if current > lektricoMaxCurrentA {
		current = lektricoMaxCurrentA
	}
	l.current.Store(current)
	return l.sendCurrent(current)
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

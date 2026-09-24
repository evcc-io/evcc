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

// dé (depow) chargers via the local Tuya protocol, Tuya model e1krltgk
// (product ids gxrtu5vljdthtd3g, witok7vhhjohtr02). Not all firmware versions report all data points.
//
// Sources: Tuya model definitions in make-all/tuya-local#3120, #3952 and Apollon77/ioBroker.tuya#747,
// tuya-local dewall_evcharger.yaml, vondraussen/de-wallbox-evcc-gateway.
//
//	DP   code                access  name (translated)           notes
//	101  x_work_state        rw      work state                  100 offline, 101 no vehicle, 200 vehicle connected, 201 charging complete (disconnect required),
//	                                                             202 waiting (schedule), 203 waiting (delay), 204 paused, 300 charging,
//	                                                             400 overcurrent, 401 overvoltage, 402 undervoltage, 403 overtemperature,
//	                                                             500 self test failed, 501 residual current, 502 relay welded,
//	                                                             503 residual current self test failed, 504 control pilot error,
//	                                                             505 other error, 506 diode failure, 507 earthing protection
//	102  x_metrics           ro      metrics                     {"L1":[V,A,P],"L2":[...],"L3":[...],"t":280,"p":39,"d":20110,"e":22}
//	                                                             V 0.1 V, A 0.1 A, P 0.1 kW, t 0.1 °C, p 0.1 kW, d session s, e session 0.1 kWh
//	103  x_selftest          ro      power-on self test result   unused according to vendor
//	104  x_alarm             ro      alarm                       {"t":"2025-10-23 21:30:00","v":<error code>}
//	105  x_charge_history    ro      charge history              last session {"t":"2026-09-05 13:15:20","s":"13:15","e":"10:50","d":77732,"c":62}, c 0.1 kWh
//	106  x_charger_info      rw      device info                 {"r":"Type B, AC 30mA + DC 6mA","fv":"2.9.3","cp":"11.5","t":"0","e":"0"}
//	                                                             r RCD type, fv firmware, cp control pilot voltage
//	107  x_adjust_current    ro      selectable currents         "[6, 8, 10, 13, 16]"
//	108  x_downcounter       ro      countdown remaining         s
//	109  x_work_st_debug     ro      work state (debug)          SLEEP, IDLE, IDLEINS, WORKING, WAIT, ERRORPAUSE, PAUSE, STOP, EMPTY
//	110  x_single_fase_mode  rw      single/three phase mode     bool, not seen on firmware 2.9.3
//	111  x_debug             rw      spare                       string
//	140  x_do_charge         wr      start/stop session          write-only trigger, true start, false stop. Not used: sessions are started by the device
//	141  x_do_reset          wr      factory reset               write-only trigger
//	142  x_do_reboot         wr      reboot                      write-only trigger
//	150  x_charge_current    rw      charging current            A, 0 pauses the vehicle via control pilot (not in model range 6-32)
//	151  x_charge_mode       rw      charging mode               {"m":0,"dt":0,"ss":"00:00","se":"08:00"}, m 0 immediate, 2 schedule ss-se, user setting
//	152  x_max_current_cfg   rw      maximum charging current    A, installation limit
//	153  x_lang_cfg          rw      language/debug config       string
//	154  x_socket_cfg        rw      earthing option             0 prompt, 1 charge directly, 2 cancel charging
//	155  x_nfc_cfg           rw      NFC                         bool
//	156  x_earch_free_cfg    rw      earth-free option           bool
//	157  x_product_varient   ro      product variant             0 default, 1 without NFC
//	188  x_heartbeat         rw      host heartbeat              true enables fast metrics updates for ~30 s
//	189  dp_num              ro      number of reported DPs
//	190  x_plug_charge       rw      plug and charge             bool, start charging when a vehicle is plugged in, user setting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/tuya"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
)

const (
	depowDpWorkState = "101"
	depowDpMetrics   = "102"
	depowDpSteps     = "107"
	depowDpCurrent   = "150"
	depowDpNfc       = "155"
	depowDpRefresh   = "188"

	depowRefreshInterval = 25 * time.Second
)

var depowDefaultSteps = []int64{6, 8, 10, 13, 16}

type depowMetrics struct {
	L1, L2, L3 [3]float64
	P          float64 `json:"p"`
	E          float64 `json:"e"`
}

// Depow charger implementation
type Depow struct {
	log  *util.Logger
	conn *tuya.Connection

	mu        sync.Mutex
	current   int64
	refreshed time.Time
}

func init() {
	registry.AddCtx("depow", NewDepowFromConfig)
}

// NewDepowFromConfig creates a dé charger from generic config
func NewDepowFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	var cc struct {
		Host     string
		Id       string
		LocalKey string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Host == "" {
		return nil, errors.New("missing host")
	}

	if cc.Id == "" || cc.LocalKey == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewDepow(ctx, cc.Host, cc.Id, cc.LocalKey)
}

// NewDepow creates a dé charger
func NewDepow(ctx context.Context, host, id, localKey string) (_ *Depow, err error) {
	log := util.NewLogger("depow").Redact(localKey)

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	// stop reconnecting if the device is not reachable during setup
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	conn, err := tuya.NewConnection(ctx, log, host, id, localKey)
	if err != nil {
		return nil, err
	}

	dps, err := conn.DpsContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("device not reachable: %w", err)
	}

	wb := &Depow{
		log:     log,
		conn:    conn,
		current: depowSteps(dps)[0],
	}

	if current := depowInt(dps[depowDpCurrent]); current > 0 {
		wb.current = current
	}

	return wb, nil
}

func depowInt(v any) int64 {
	f, _ := v.(float64)
	return int64(f)
}

func depowSteps(dps map[string]any) []int64 {
	if s, ok := dps[depowDpSteps].(string); ok {
		var res []int64
		if err := json.Unmarshal([]byte(s), &res); err == nil && len(res) > 0 {
			return slices.Sorted(slices.Values(res))
		}
	}
	return depowDefaultSteps
}

// depowStep returns the highest supported current not exceeding the requested current
func depowStep(steps []int64, current int64) int64 {
	res := steps[0]
	for _, s := range steps {
		if s <= current {
			res = s
		}
	}
	return res
}

// Status implements the api.Charger interface
func (wb *Depow) Status() (api.ChargeStatus, error) {
	dps, err := wb.conn.Dps()
	if err != nil {
		return api.StatusNone, err
	}

	return depowStatus(dps[depowDpWorkState])
}

func depowStatus(v any) (api.ChargeStatus, error) {
	code, ok := v.(float64)
	if !ok {
		return api.StatusNone, fmt.Errorf("invalid work state: %v", v)
	}

	switch state := int(code); state / 100 {
	case 1:
		return api.StatusA, nil
	case 2:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid work state: %d", state)
	}
}

var _ api.StatusReasoner = (*Depow)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *Depow) StatusReason() (api.Reason, error) {
	dps, err := wb.conn.Dps()
	if err != nil {
		return api.ReasonUnknown, err
	}

	switch depowInt(dps[depowDpWorkState]) {
	case 200:
		if nfc, _ := dps[depowDpNfc].(bool); nfc {
			return api.ReasonWaitingForAuthorization, nil
		}
	case 201:
		return api.ReasonDisconnectRequired, nil
	}

	return api.ReasonUnknown, nil
}

// Enabled implements the api.Charger interface
func (wb *Depow) Enabled() (bool, error) {
	dps, err := wb.conn.Dps()
	return depowInt(dps[depowDpCurrent]) > 0, err
}

// Enable implements the api.Charger interface
func (wb *Depow) Enable(enable bool) error {
	var current int64
	if enable {
		wb.mu.Lock()
		current = wb.current
		wb.mu.Unlock()
	}

	return wb.conn.Set(map[string]any{depowDpCurrent: current})
}

// MaxCurrent implements the api.Charger interface
func (wb *Depow) MaxCurrent(current int64) error {
	dps, err := wb.conn.Dps()
	if err != nil {
		return err
	}

	step := depowStep(depowSteps(dps), current)

	wb.mu.Lock()
	wb.current = step
	wb.mu.Unlock()

	// current 0 means disabled, keep until enabled
	if actual := depowInt(dps[depowDpCurrent]); actual == 0 || actual == step {
		return nil
	}

	return wb.conn.Set(map[string]any{depowDpCurrent: step})
}

func (wb *Depow) metrics() (depowMetrics, error) {
	var res depowMetrics

	wb.mu.Lock()
	if time.Since(wb.refreshed) > depowRefreshInterval {
		if err := wb.conn.Set(map[string]any{depowDpRefresh: true}); err == nil {
			wb.refreshed = time.Now()
		}
	}
	wb.mu.Unlock()

	dps, err := wb.conn.Dps()
	if err != nil {
		return res, err
	}

	s, ok := dps[depowDpMetrics].(string)
	if !ok {
		return res, api.ErrNotAvailable
	}

	err = json.Unmarshal([]byte(s), &res)
	return res, err
}

var _ api.Meter = (*Depow)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Depow) CurrentPower() (float64, error) {
	res, err := wb.metrics()
	return res.P * 100, err
}

var _ api.ChargeRater = (*Depow)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Depow) ChargedEnergy() (float64, error) {
	res, err := wb.metrics()
	return res.E / 10, err
}

var _ api.PhaseCurrents = (*Depow)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Depow) Currents() (float64, float64, float64, error) {
	res, err := wb.metrics()
	return res.L1[1] / 10, res.L2[1] / 10, res.L3[1] / 10, err
}

var _ api.PhaseVoltages = (*Depow)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Depow) Voltages() (float64, float64, float64, error) {
	res, err := wb.metrics()
	return res.L1[0] / 10, res.L2[0] / 10, res.L3[0] / 10, err
}

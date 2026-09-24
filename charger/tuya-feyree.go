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

// Chargers sharing the DP layout used by Shenzhen Feyree Technology and Xinxiang Kolanky Technical (iSIGMA),
// also sold as Absina, dé 22.2 kW, EcoPower and Goodcell, via the local Tuya protocol.
// Tuya model ids differ per product, e.g. ek3pa0 for product id tj9l3ghsjnbdjom6. Not all firmware versions report all data points.
//
// Sources: Tuya model definition in make-all/tuya-local#2050, DP dumps in make-all/tuya-local#1570, #1853, #2628, #4632, #6197,
// tuya-local absina_evcharger.yaml, feyree_ev_portable_charger.yaml, feyree_43_evcharger.yaml, goodcell_ev_charger.yaml, kolanky_evcharger.yaml.
//
//	DP   code                access  name (translated)           notes
//	3    work_state          ro      work state                  charger_free, charger_insert, charger_free_fault, charger_wait,
//	                                                             charger_charging, charger_pause, charger_end, charger_fault
//	10   fault               ro      fault alarm                 bitmap err_uvp, err_ovp, err_ocp, err_pe, err_temp, err_cp, err_leak, err_leaksc,
//	                                                             err_pe2, err_temp_plug, err_temp_pcb, err_temp_core, err_esb, err_pe_sck
//	11   alarm_set_1         rw      alarm settings 1            raw
//	12   alarm_set_2         rw      alarm settings 2            raw
//	14   work_mode           rw      work mode                   charge_now, charge_pct, charge_energy, charge_schedule
//	15   balance_energy      ro      remaining energy            0.001 kWh
//	16   clear_energy        rw      clear energy                bool
//	18   switch              rw      switch                      factory reset according to tuya-local, never written
//	23   system_version      ro      system version              "HW V1.0,SW V1.0.3"
//	25   charge_energy_once  ro      single charge energy        0.01 kWh
//	27   online_state        rw      online state                online, offline, enables real time updates according to tuya-local
//	101  DeviceState         ro      device state                no_connet, connect, charing, wait_rfid, finish, wait_charing, error
//	102  A_Voltage           ro      voltage L1                  V, some firmware 0.1 V
//	103  B_Voltage           ro      voltage L2                  V, some firmware 0.1 V
//	104  C_Voltage           ro      voltage L3                  V, some firmware 0.1 V
//	105  A_Current           ro      current L1                  0.1 A
//	106  B_Current           ro      current L2                  0.1 A
//	107  C_Current           ro      current L3                  0.1 A
//	108  PhaseFlag           ro      single/three phase          Single_phase, Three_phase, No_phase, Phase_err, unreliable
//	109  DeviceKw            ro      power                       0.1 kW
//	110  DeviceTemp          ro      temperature                 0.1 °C
//	111  DeviceTemp2         ro      temperature 2               0.1 °C
//	112  DeviceKwh           ro      session energy              0.1 kWh, reset on unplug
//	113  DeviceMaxSetA       ro      maximum current setting     Max16A, Max32A, Max40A, Max50A, selects the current DP
//	114  Set16A              rw      charging current            A, 6/8-16
//	115  Set32A              rw      charging current            A, 6/8-32
//	116  Set40A              rw      charging current            A, 8/12-40
//	117  Set50A              rw      charging current            A, 8/12-50
//	118  SetDelayTime        rw      delayed charging            h, 0-15
//	119  SetDefineTime       rw      timed charging              h, 0-15
//	120  Ctime               ro      time                        "00:00:00"
//	121  CTime2              ro      charging time               0.1 h
//	122  IDVerificationSet   rw      identity verification       bool, require RFID
//	123  RFID                rw      card swipe                  bool
//	124  ChargingOperation   rw      charging operation          OpenCharging, CloseCharging, WaitOperation. Not used: sessions are started by the device

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/tuya"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
)

const (
	tuyaFeyreeDpState      = "101"
	tuyaFeyreeDpPower      = "109"
	tuyaFeyreeDpEnergy     = "112"
	tuyaFeyreeDpMaxSetting = "113"
)

var (
	tuyaFeyreeDpVoltages = [3]string{"102", "103", "104"}
	tuyaFeyreeDpCurrents = [3]string{"105", "106", "107"}
)

type tuyaFeyreeCurrent struct {
	dp       string
	min, max int64
}

// current setpoint data point and limits by maximum current setting.
// Minimum currents are 6/6/8/8 A in tuya-local configs and 8/8/12/12 A in the Tuya model, 6 A is confirmed for Max16A.
var tuyaFeyreeCurrents = map[string]tuyaFeyreeCurrent{
	"Max16A": {"114", 6, 16},
	"Max32A": {"115", 6, 32},
	"Max40A": {"116", 8, 40},
	"Max50A": {"117", 8, 50},
}

// TuyaFeyree charger implementation
type TuyaFeyree struct {
	log     *util.Logger
	conn    *tuya.Connection
	setting tuyaFeyreeCurrent

	mu      sync.Mutex
	current int64
}

func init() {
	registry.AddCtx("tuya-feyree", NewTuyaFeyreeFromConfig)
}

// NewTuyaFeyreeFromConfig creates a Feyree charger from generic config
func NewTuyaFeyreeFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
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

	return NewTuyaFeyree(ctx, cc.Host, cc.Id, cc.LocalKey)
}

// NewTuyaFeyree creates a Feyree charger
func NewTuyaFeyree(ctx context.Context, host, id, localKey string) (_ *TuyaFeyree, err error) {
	log := util.NewLogger("tuya-feyree").Redact(localKey)

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

	setting, err := tuyaFeyreeCurrentSetting(dps)
	if err != nil {
		return nil, err
	}

	wb := &TuyaFeyree{
		log:     log,
		conn:    conn,
		setting: setting,
		current: setting.min,
	}

	if current := int64(tuyaFeyreeFloat(dps[setting.dp])); current > 0 {
		wb.current = current
	}

	return wb, nil
}

// tuyaFeyreeCurrentSetting returns the current setpoint data point and limits matching the maximum current setting,
// or the first current data point reported by the device
func tuyaFeyreeCurrentSetting(dps map[string]any) (tuyaFeyreeCurrent, error) {
	if s, ok := dps[tuyaFeyreeDpMaxSetting].(string); ok {
		if res, ok := tuyaFeyreeCurrents[s]; ok {
			return res, nil
		}
	}

	for _, s := range []string{"Max16A", "Max32A", "Max40A", "Max50A"} {
		if res := tuyaFeyreeCurrents[s]; dps[res.dp] != nil {
			return res, nil
		}
	}

	return tuyaFeyreeCurrent{}, fmt.Errorf("unknown maximum current setting: %v", dps[tuyaFeyreeDpMaxSetting])
}

func tuyaFeyreeFloat(v any) float64 {
	f, _ := v.(float64)
	return f
}

// tuyaFeyreeValue returns a numeric data point or api.ErrNotAvailable if the device does not report it
func tuyaFeyreeValue(dps map[string]any, dp string) (float64, error) {
	f, ok := dps[dp].(float64)
	if !ok {
		return 0, api.ErrNotAvailable
	}
	return f, nil
}

// Status implements the api.Charger interface
func (wb *TuyaFeyree) Status() (api.ChargeStatus, error) {
	dps, err := wb.conn.Dps()
	if err != nil {
		return api.StatusNone, err
	}

	return tuyaFeyreeStatus(dps[tuyaFeyreeDpState])
}

func tuyaFeyreeStatus(state any) (api.ChargeStatus, error) {
	switch state {
	case "no_connet":
		return api.StatusA, nil
	case "connect", "wait_rfid", "wait_charing", "finish":
		return api.StatusB, nil
	case "charing":
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid device state: %v", state)
	}
}

var _ api.StatusReasoner = (*TuyaFeyree)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *TuyaFeyree) StatusReason() (api.Reason, error) {
	dps, err := wb.conn.Dps()
	if err != nil {
		return api.ReasonUnknown, err
	}

	switch dps[tuyaFeyreeDpState] {
	case "wait_rfid":
		return api.ReasonWaitingForAuthorization, nil
	case "finish":
		return api.ReasonDisconnectRequired, nil
	default:
		return api.ReasonUnknown, nil
	}
}

// Enabled implements the api.Charger interface
func (wb *TuyaFeyree) Enabled() (bool, error) {
	dps, err := wb.conn.Dps()
	return tuyaFeyreeFloat(dps[wb.setting.dp]) > 0, err
}

// Enable implements the api.Charger interface
func (wb *TuyaFeyree) Enable(enable bool) error {
	var current int64
	if enable {
		wb.mu.Lock()
		current = wb.current
		wb.mu.Unlock()
	}

	return wb.conn.Set(map[string]any{wb.setting.dp: current})
}

// MaxCurrent implements the api.Charger interface
func (wb *TuyaFeyree) MaxCurrent(current int64) error {
	dps, err := wb.conn.Dps()
	if err != nil {
		return err
	}

	current = min(max(current, wb.setting.min), wb.setting.max)

	wb.mu.Lock()
	wb.current = current
	wb.mu.Unlock()

	// current 0 means disabled, keep until enabled
	if actual := int64(tuyaFeyreeFloat(dps[wb.setting.dp])); actual == 0 || actual == current {
		return nil
	}

	return wb.conn.Set(map[string]any{wb.setting.dp: current})
}

var _ api.CurrentLimiter = (*TuyaFeyree)(nil)

// GetMinMaxCurrent implements the api.CurrentLimiter interface
func (wb *TuyaFeyree) GetMinMaxCurrent() (float64, float64, error) {
	return float64(wb.setting.min), float64(wb.setting.max), nil
}

var _ api.Meter = (*TuyaFeyree)(nil)

// CurrentPower implements the api.Meter interface
func (wb *TuyaFeyree) CurrentPower() (float64, error) {
	dps, err := wb.conn.Dps()
	if err != nil {
		return 0, err
	}

	res, err := tuyaFeyreeValue(dps, tuyaFeyreeDpPower)
	return res * 100, err
}

var _ api.ChargeRater = (*TuyaFeyree)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *TuyaFeyree) ChargedEnergy() (float64, error) {
	dps, err := wb.conn.Dps()
	if err != nil {
		return 0, err
	}

	res, err := tuyaFeyreeValue(dps, tuyaFeyreeDpEnergy)
	return res / 10, err
}

// phases returns the scaled values of three data points. L2 and L3 may be missing on single phase devices.
func (wb *TuyaFeyree) phases(dp [3]string, scale func(float64) float64) (float64, float64, float64, error) {
	dps, err := wb.conn.Dps()
	if err != nil {
		return 0, 0, 0, err
	}

	l1, err := tuyaFeyreeValue(dps, dp[0])
	if err != nil {
		return 0, 0, 0, err
	}

	return scale(l1), scale(tuyaFeyreeFloat(dps[dp[1]])), scale(tuyaFeyreeFloat(dps[dp[2]])), nil
}

var _ api.PhaseCurrents = (*TuyaFeyree)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *TuyaFeyree) Currents() (float64, float64, float64, error) {
	return wb.phases(tuyaFeyreeDpCurrents, func(v float64) float64 { return v / 10 })
}

var _ api.PhaseVoltages = (*TuyaFeyree)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *TuyaFeyree) Voltages() (float64, float64, float64, error) {
	return wb.phases(tuyaFeyreeDpVoltages, tuyaFeyreeVoltage)
}

// tuyaFeyreeVoltage scales voltages, the model range is 0-500 V but some firmware reports 0.1 V
func tuyaFeyreeVoltage(v float64) float64 {
	if v > 500 {
		return v / 10
	}
	return v
}

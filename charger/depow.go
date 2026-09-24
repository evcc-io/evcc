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

// dé (depow) chargers via the local Tuya protocol.
// Data points: https://github.com/make-all/tuya-local/blob/main/custom_components/tuya_local/devices/dewall_evcharger.yaml
//
//	102 metrics  {"L1":[2240,58,13],"L2":[...],"L3":[...],"t":280,"p":39,"d":20110,"e":22}
//	             L = [0.1 V, 0.1 A, 0.1 kW], p = 0.1 kW, e = session 0.1 kWh
//	107 steps    "[6, 8, 10, 13, 16]"
//	109 status   SLEEP, IDLE, IDLEINS, WORKING, WAIT, ERRORPAUSE, PAUSE, STOP
//	140 charge   true = start, false = stop
//	150 current  A
//	151 mode     {"m":0,"dt":0,"ss":"00:00","se":"08:00"}
//	188 refresh  triggers fast metrics updates for ~30s

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
	depowDpMetrics = "102"
	depowDpSteps   = "107"
	depowDpStatus  = "109"
	depowDpCharge  = "140"
	depowDpCurrent = "150"
	depowDpMode    = "151"
	depowDpRefresh = "188"

	depowRefreshInterval = 25 * time.Second
	depowLockMargin      = 6 * time.Hour
)

var depowDefaultSteps = []int64{6, 8, 10, 13, 16}

type depowMode struct {
	M  int    `json:"m"`
	Dt int    `json:"dt"`
	Ss string `json:"ss"`
	Se string `json:"se"`
}

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
	enabled   bool
	lockSent  time.Time
	refreshed time.Time
}

func init() {
	registry.AddCtx("depow", NewDepowFromConfig)
}

// NewDepowFromConfig creates a dé charger from generic config
func NewDepowFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		Host     string
		Id       string
		LocalKey string
		Version  string
	}{
		Version: "3.5",
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

	return NewDepow(ctx, cc.Host, cc.Id, cc.LocalKey, cc.Version)
}

// NewDepow creates a dé charger
func NewDepow(ctx context.Context, host, id, localKey, version string) (_ *Depow, err error) {
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

	conn, err := tuya.NewConnection(ctx, log, host, id, localKey, version)
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
		enabled: dps[depowDpStatus] == "WORKING",
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Depow) Status() (api.ChargeStatus, error) {
	dps, err := wb.conn.Dps()
	if err != nil {
		return api.StatusNone, err
	}

	status, _ := dps[depowDpStatus].(string)

	if err := wb.lock(dps, status); err != nil {
		wb.log.WARN.Printf("scheduler lock: %v", err)
	}

	switch status {
	case "SLEEP", "IDLE":
		return api.StatusA, nil
	case "IDLEINS", "WAIT", "PAUSE", "STOP":
		return api.StatusB, nil
	case "WORKING":
		return api.StatusC, nil
	case "ERRORPAUSE":
		return api.StatusNone, errors.New("charger fault")
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %v", dps[depowDpStatus])
	}
}

// lock keeps a disabled charger from starting automatically when a vehicle is plugged in.
// The scheduler window is kept far ahead of the current time so it is never reached.
func (wb *Depow) lock(dps map[string]any, status string) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if wb.enabled || status == "WORKING" || time.Since(wb.lockSent) < time.Minute {
		return nil
	}

	now := time.Now()

	var mode depowMode
	if s, ok := dps[depowDpMode].(string); ok && json.Unmarshal([]byte(s), &mode) == nil && mode.M == 2 {
		if start, err := time.Parse("15:04", mode.Ss); err == nil {
			day := 24 * time.Hour
			ahead := (time.Duration(start.Hour()-now.Hour())*time.Hour + time.Duration(start.Minute()-now.Minute())*time.Minute + day) % day
			if ahead >= depowLockMargin {
				return nil
			}
		}
	}

	start := now.Add(12 * time.Hour)
	b, err := json.Marshal(depowMode{M: 2, Dt: 8, Ss: start.Format("15:04"), Se: start.Add(time.Minute).Format("15:04")})
	if err != nil {
		return err
	}

	wb.lockSent = now
	return wb.conn.Set(map[string]any{depowDpMode: string(b)})
}

// Enabled implements the api.Charger interface
func (wb *Depow) Enabled() (bool, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	return wb.enabled, nil
}

// Enable implements the api.Charger interface
func (wb *Depow) Enable(enable bool) error {
	dps := map[string]any{depowDpCharge: enable}

	if enable {
		b, err := json.Marshal(depowMode{Ss: "00:00", Se: "08:00"})
		if err != nil {
			return err
		}
		dps[depowDpMode] = string(b)
	}

	if err := wb.conn.Set(dps); err != nil {
		return err
	}

	wb.mu.Lock()
	wb.enabled = enable
	wb.lockSent = time.Time{}
	wb.mu.Unlock()

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *Depow) MaxCurrent(current int64) error {
	dps, err := wb.conn.Dps()
	if err != nil {
		return err
	}

	steps := depowDefaultSteps
	if s, ok := dps[depowDpSteps].(string); ok {
		var res []int64
		if err := json.Unmarshal([]byte(s), &res); err == nil && len(res) > 0 {
			steps = res
		}
	}

	step := depowStep(steps, current)
	if v, ok := dps[depowDpCurrent].(float64); ok && int64(v) == step {
		return nil
	}

	return wb.conn.Set(map[string]any{depowDpCurrent: step})
}

// depowStep returns the highest supported current not exceeding the requested current
func depowStep(steps []int64, current int64) int64 {
	steps = slices.Sorted(slices.Values(steps))

	res := steps[0]
	for _, s := range steps {
		if s <= current {
			res = s
		}
	}

	return res
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

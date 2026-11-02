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
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ecoflow"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
)

// ecoflowSettle is how long after a command heartbeats may still show the
// state before it. On hardware a start showed after up to 44 s, a current after 40 s.
const ecoflowSettle = time.Minute

// EcoflowPowerPulse is an EcoFlow PowerPulse wallbox behind a PowerOcean
// gateway, controlled through the EcoFlow app MQTT broker.
type EcoflowPowerPulse struct {
	wb *ecoflow.Wallbox

	mu        sync.Mutex
	enabled   bool
	enabledAt time.Time
	current   float64
	currentAt time.Time
}

func init() {
	registry.AddCtx("ecoflow-powerpulse", NewEcoflowPowerPulseFromConfig)
}

// NewEcoflowPowerPulseFromConfig creates an EcoFlow PowerPulse charger from generic config
func NewEcoflowPowerPulseFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		User, Password  string
		Serial, Gateway string
		Timeout         time.Duration
	}{
		Timeout: 3 * time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.Serial == "" || cc.Gateway == "" {
		return nil, errors.New("missing serial or gateway")
	}

	conn, err := ecoflow.NewConnection(cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	wb, err := conn.Wallbox(ctx, cc.Serial, cc.Gateway, cc.Timeout)
	if err != nil {
		return nil, err
	}

	return &EcoflowPowerPulse{wb: wb}, nil
}

// Status implements the api.Charger interface
func (c *EcoflowPowerPulse) Status() (api.ChargeStatus, error) {
	hb, err := c.wb.Heartbeat()
	if err != nil {
		return api.StatusNone, err
	}

	switch hb.State {
	case ecoflow.StateUnplugged:
		return api.StatusA, nil
	case ecoflow.StatePlugged, ecoflow.StatePaused, ecoflow.StatePausedByVehicle, ecoflow.StateStoppedByWallbox:
		return api.StatusB, nil
	case ecoflow.StateCharging:
		return api.StatusC, nil
	case ecoflow.StateError:
		return api.StatusNone, errors.New("wallbox fault (state 9)")
	default:
		return api.StatusNone, fmt.Errorf("unknown state: %d", hb.State)
	}
}

// Enabled implements the api.Charger interface. A plugged, idle wallbox may
// be released or not, so that reports the last command.
func (c *EcoflowPowerPulse) Enabled() (bool, error) {
	hb, err := c.wb.Heartbeat()
	if err != nil {
		return false, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if hb.Received.Sub(c.enabledAt) >= ecoflowSettle {
		switch {
		case hb.State.Enabled():
			return true, nil
		case hb.State == ecoflow.StateStoppedByWallbox:
			return false, nil
		}
	}

	return c.enabled, nil
}

// Enable implements the api.Charger interface
func (c *EcoflowPowerPulse) Enable(enable bool) error {
	if err := c.wb.StartStop(enable); err != nil {
		return err
	}

	c.mu.Lock()
	c.enabled, c.enabledAt = enable, time.Now()
	c.mu.Unlock()

	return nil
}

// MaxCurrent implements the api.Charger interface
func (c *EcoflowPowerPulse) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*EcoflowPowerPulse)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *EcoflowPowerPulse) MaxCurrentMillis(current float64) error {
	raw := uint64(math.Round(current * 10))
	if err := c.wb.SetCurrent(raw); err != nil {
		return err
	}

	c.mu.Lock()
	c.current, c.currentAt = float64(raw)/10, time.Now()
	c.mu.Unlock()

	return nil
}

var _ api.CurrentGetter = (*EcoflowPowerPulse)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (c *EcoflowPowerPulse) GetMaxCurrent() (float64, error) {
	hb, err := c.wb.Heartbeat()
	if err != nil {
		return 0, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if hb.ChargeCurrentRaw > 0 && hb.Received.Sub(c.currentAt) >= ecoflowSettle {
		return float64(hb.ChargeCurrentRaw) / 10, nil
	}

	if c.current == 0 {
		return 0, api.ErrNotAvailable
	}

	return c.current, nil
}

var _ api.CurrentLimiter = (*EcoflowPowerPulse)(nil)

// GetMinMaxCurrent implements the api.CurrentLimiter interface. The wallbox
// does not store a session current above its configured maximum.
func (c *EcoflowPowerPulse) GetMinMaxCurrent() (float64, float64, error) {
	hb, err := c.wb.Heartbeat()
	if err != nil || hb.MaxCurrentRaw == 0 {
		return 0, 0, api.ErrNotAvailable
	}

	return 0, float64(hb.MaxCurrentRaw) / 10, nil
}

var _ api.PhaseSwitcher = (*EcoflowPowerPulse)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface. The wallbox only
// switches while stopped; the loadpoint starts it again after the switch.
func (c *EcoflowPowerPulse) Phases1p3p(phases int) error {
	var mode uint64 = ecoflow.PhaseThree
	if phases == 1 {
		mode = ecoflow.PhaseOne
	}

	if hb, err := c.wb.Heartbeat(); err == nil && hb.State == ecoflow.StateCharging {
		if err := c.Enable(false); err != nil {
			return err
		}

		if err := c.waitStopped(); err != nil {
			return err
		}
	}

	return c.wb.SetPhaseMode(mode)
}

// waitStopped waits for a heartbeat showing the stop, which took 2-3 s on hardware.
func (c *EcoflowPowerPulse) waitStopped() error {
	for range 30 {
		time.Sleep(time.Second)
		if hb, err := c.wb.Heartbeat(); err == nil && hb.State != ecoflow.StateCharging {
			return nil
		}
	}

	return errors.New("stop not confirmed, phases not switched")
}

var _ api.Meter = (*EcoflowPowerPulse)(nil)

// CurrentPower implements the api.Meter interface
func (c *EcoflowPowerPulse) CurrentPower() (float64, error) {
	hb, err := c.wb.Heartbeat()
	if err != nil {
		return 0, err
	}

	return float64(hb.PowerW), nil
}

var _ api.PhaseCurrents = (*EcoflowPowerPulse)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *EcoflowPowerPulse) Currents() (float64, float64, float64, error) {
	hb, err := c.wb.Heartbeat()
	if err != nil {
		return 0, 0, 0, err
	}

	return ecoflowPhases("currents", hb.Amps)
}

var _ api.PhaseVoltages = (*EcoflowPowerPulse)(nil)

// Voltages implements the api.PhaseVoltages interface
func (c *EcoflowPowerPulse) Voltages() (float64, float64, float64, error) {
	hb, err := c.wb.Heartbeat()
	if err != nil {
		return 0, 0, 0, err
	}

	return ecoflowPhases("voltages", hb.Volts)
}

func ecoflowPhases(name string, v []float32) (float64, float64, float64, error) {
	if len(v) != 3 {
		return 0, 0, 0, fmt.Errorf("%s: expected 3 phases, got %d", name, len(v))
	}

	return float64(v[0]), float64(v[1]), float64(v[2]), nil
}

var _ api.MeterEnergy = (*EcoflowPowerPulse)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *EcoflowPowerPulse) TotalEnergy() (float64, error) {
	hb, err := c.wb.Heartbeat()
	if err != nil {
		return 0, err
	}

	if hb.TotalEnergyWh == 0 {
		return 0, api.ErrNotAvailable
	}

	return float64(hb.TotalEnergyWh) / 1e3, nil
}

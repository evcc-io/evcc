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
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/grid"
	"github.com/samber/lo"
)

// StagedHeater controls a heating rod with up to three equal stages, each switched by its own
// relay on its own phase, e.g. three single-phase switch sockets. Stages are switched cumulatively.
type StagedHeater struct {
	*embed
	implement.Caps
	log        *util.Logger
	relays     []api.Charger
	stepPower  float64
	hysteresis float64

	mu      sync.Mutex
	lp      loadpoint.API
	on      []bool  // last known relay states
	target  int     // stages last switched
	current float64 // last requested current
}

// stagedHeaterMinStepPower keeps a deviating stage current above the loadpoint's sync threshold of 0.23A
const stagedHeaterMinStepPower = 500

func init() {
	registry.AddCtx("staged-heater", NewStagedHeaterFromConfig)
}

// NewStagedHeaterFromConfig creates a staged heater from generic config
func NewStagedHeaterFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		embed                  `mapstructure:",squash"`
		Stage1, Stage2, Stage3 *config.Typed
		StepPower              float64
		Hysteresis             float64
	}{
		embed: embed{
			Icon_:     "waterheater",
			Features_: []api.Feature{api.Heating, api.IntegratedDevice},
		},
		Hysteresis: 300,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// stages are named fields like sgready-relay, so static `type: device` references are initialized first
	stages := []*config.Typed{cc.Stage1, cc.Stage2, cc.Stage3}
	for len(stages) > 0 && stages[len(stages)-1] == nil {
		stages = stages[:len(stages)-1]
	}

	relays := make([]api.Charger, 0, len(stages))
	for i, stage := range stages {
		if stage == nil {
			return nil, fmt.Errorf("stage %d: missing relay", i+1)
		}
		if name, _ := stage.Other["name"].(string); stage.Type == "" || stage.Type == "device" && name == "" {
			return nil, fmt.Errorf("stage %d: missing relay", i+1)
		}
		if slices.ContainsFunc(stages[:i], func(s *config.Typed) bool {
			return strings.EqualFold(s.Type, stage.Type) && reflect.DeepEqual(s.Other, stage.Other)
		}) {
			return nil, fmt.Errorf("stage %d: duplicate relay", i+1)
		}

		relay, err := NewFromConfig(ctx, stage.Type, stage.Other)
		if err != nil {
			return nil, fmt.Errorf("stage %d: %w", i+1, err)
		}
		relays = append(relays, relay)
	}

	return NewStagedHeater(&cc.embed, relays, cc.StepPower, cc.Hysteresis)
}

// NewStagedHeater creates a staged heater from its stage relays. Power, energy and phase
// currents are provided if all relays provide them.
func NewStagedHeater(embed *embed, relays []api.Charger, stepPower, hysteresis float64) (*StagedHeater, error) {
	if len(relays) < 1 || len(relays) > 3 {
		return nil, fmt.Errorf("invalid number of stages: %d (1-3)", len(relays))
	}
	if stepPower < stagedHeaterMinStepPower {
		return nil, fmt.Errorf("invalid step power: %g (min %dW)", stepPower, stagedHeaterMinStepPower)
	}
	if hysteresis < 0 || hysteresis >= stepPower {
		return nil, fmt.Errorf("invalid hysteresis: %g", hysteresis)
	}

	wb := &StagedHeater{
		embed:      embed,
		Caps:       implement.New(),
		log:        util.NewLogger("staged-heater"),
		relays:     relays,
		stepPower:  stepPower,
		hysteresis: hysteresis,
		on:         make([]bool, len(relays)),
	}

	// adopt a running heater
	for i, r := range relays {
		on, err := r.Enabled()
		if err != nil {
			return nil, fmt.Errorf("stage %d: %w", i+1, err)
		}
		wb.on[i] = on
	}
	wb.target = wb.stages()

	if meters, ok := stageCaps[api.Meter](relays); ok {
		implement.Has(wb, implement.Meter(func() (float64, error) {
			res, err := wb.readings(func(i int) (float64, error) { return meters[i].CurrentPower() }, stepPower)
			return res[0] + res[1] + res[2], err
		}))
	}

	if meters, ok := stageCaps[api.MeterEnergy](relays); ok {
		implement.Has(wb, implement.MeterEnergy(func() (float64, error) {
			var res float64
			for i, m := range meters {
				energy, err := m.TotalEnergy()
				if err != nil {
					return 0, fmt.Errorf("stage %d: %w", i+1, err)
				}
				res += energy
			}
			return res, nil
		}))
	}

	// stage 1-3 is phase L1-L3
	if meters, ok := stageCaps[api.PhaseCurrents](relays); ok {
		implement.Has(wb, implement.PhaseCurrents(func() (float64, float64, float64, error) {
			res, err := wb.readings(func(i int) (float64, error) {
				l1, l2, l3, err := meters[i].Currents()
				return l1 + l2 + l3, err
			}, grid.PowerToCurrent(stepPower, 1))
			return res[0], res[1], res[2], err
		}))
	}

	return wb, nil
}

// stageCaps returns the capability of all relays, or false if any relay lacks it
func stageCaps[T any](relays []api.Charger) ([]T, bool) {
	res := make([]T, 0, len(relays))
	for _, r := range relays {
		c, ok := api.Cap[T](r)
		if !ok {
			return nil, false
		}
		res = append(res, c)
	}
	return res, true
}

// stages returns the number of active relays
func (wb *StagedHeater) stages() int {
	return lo.Count(wb.on, true)
}

func (wb *StagedHeater) stagesForPower(power float64, active int) int {
	const eps = 1e-9 // float conversion at exact stage boundaries

	n := int(math.Floor(power/wb.stepPower + eps))
	if active > 0 && n > active && n < len(wb.relays) {
		n = max(active, int(math.Floor((power-wb.hysteresis)/wb.stepPower+eps)))
	}

	return min(max(n, 1), len(wb.relays))
}

// setStages switches the first n relays on and the others off. Switching off continues
// on errors to shed as much load as possible, switching on stops to keep stages cumulative.
func (wb *StagedHeater) setStages(n int) error {
	wb.target = n

	var errs []error
	for i := len(wb.relays) - 1; i >= n; i-- {
		if wb.on[i] {
			if err := wb.relays[i].Enable(false); err != nil {
				errs = append(errs, err)
				continue
			}
			wb.on[i] = false
		}
	}

	if err := errors.Join(errs...); err != nil {
		return err
	}

	for i := range n {
		if !wb.on[i] {
			if err := wb.relays[i].Enable(true); err != nil {
				return err
			}
			wb.on[i] = true
		}
	}

	return nil
}

// Status implements the api.Charger interface. With measurement it follows the measured power,
// so an open thermostat is reported as not heating, otherwise it follows the relays.
func (wb *StagedHeater) Status() (api.ChargeStatus, error) {
	on, err := wb.Enabled()
	if m, ok := api.Cap[api.Meter](wb); ok && err == nil {
		var power float64
		power, err = m.CurrentPower()
		on = power > 50
	}

	switch {
	case err != nil:
		return api.StatusNone, err
	case on:
		return api.StatusC, nil
	default:
		return api.StatusB, nil // off or thermostat open
	}
}

// Enabled implements the api.Charger interface. Unreachable relays keep their last known
// state, so the reachable ones can still be switched off. It fails if no relay is reachable.
func (wb *StagedHeater) Enabled() (bool, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	var errs []error
	for i, r := range wb.relays {
		on, err := r.Enabled()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		wb.on[i] = on
	}

	if err := errors.Join(errs...); err != nil {
		if len(errs) == len(wb.relays) {
			return false, err
		}
		wb.log.WARN.Printf("using last known state: %v", err)
	}

	return wb.stages() > 0, nil
}

// Enable implements the api.Charger interface
func (wb *StagedHeater) Enable(enable bool) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if enable {
		return wb.setStages(wb.stagesForPower(wb.power(wb.current), 0))
	}

	// switch off every relay regardless of its last known state
	wb.target = 0

	var errs []error
	for i := len(wb.relays) - 1; i >= 0; i-- {
		if err := wb.relays[i].Enable(false); err != nil {
			errs = append(errs, err)
			continue
		}
		wb.on[i] = false
	}

	return errors.Join(errs...)
}

// power converts current to power using the active phases, or a single phase before
// the loadpoint is known, which never yields more stages than requested
func (wb *StagedHeater) power(current float64) float64 {
	phases := 1
	if wb.lp != nil {
		phases = wb.lp.ActivePhases()
	}
	return grid.CurrentToPower(current, phases)
}

// MaxCurrent implements the api.Charger interface
func (wb *StagedHeater) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*StagedHeater)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *StagedHeater) MaxCurrentMillis(current float64) error {
	if current < 0 {
		return fmt.Errorf("invalid current: %g", current)
	}

	wb.mu.Lock()
	defer wb.mu.Unlock()

	// switch only while enabled, Enable applies the current later
	if active := wb.stages(); active > 0 {
		if err := wb.setStages(wb.stagesForPower(wb.power(current), active)); err != nil {
			return err
		}
	}

	wb.current = current
	return nil
}

var _ api.CurrentGetter = (*StagedHeater)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface. If the relays deviate from the
// switched stages, e.g. switched manually, it reports 0A so the loadpoint sets the current again.
func (wb *StagedHeater) GetMaxCurrent() (float64, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	for i, on := range wb.on {
		if on != (i < wb.target) {
			return 0, nil
		}
	}

	return wb.current, nil
}

var _ api.PowerLimiter = (*StagedHeater)(nil)

// GetMinMaxPower implements the api.PowerLimiter interface
func (wb *StagedHeater) GetMinMaxPower() (float64, float64, error) {
	return wb.stepPower, wb.stepPower * float64(len(wb.relays)), nil
}

// readings returns a reading per relay. An unreachable relay is estimated from its last
// known state, so a single failure does not distort control. It fails if no relay is reachable.
func (wb *StagedHeater) readings(fun func(int) (float64, error), estimate float64) ([3]float64, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	var (
		res  [3]float64
		errs []error
	)

	for i := range wb.relays {
		v, err := fun(i)
		if err != nil {
			errs = append(errs, fmt.Errorf("stage %d: %w", i+1, err))
			if wb.on[i] {
				v = estimate
			}
		}
		res[i] = v
	}

	if err := errors.Join(errs...); err != nil {
		if len(errs) == len(wb.relays) {
			return res, err
		}
		wb.log.WARN.Printf("estimating from last known state: %v", err)
	}

	return res, nil
}

var _ loadpoint.Controller = (*StagedHeater)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *StagedHeater) LoadpointControl(lp loadpoint.API) {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	wb.lp = lp
}

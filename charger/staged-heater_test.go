package charger

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type fakeRelay struct {
	on, failSet, failGet bool
	power, energy, amps  float64
	switches             int
}

var errFake = errors.New("timeout")

func (r *fakeRelay) Enabled() (bool, error) {
	if r.failGet {
		return false, errFake
	}
	return r.on, nil
}

func (r *fakeRelay) Enable(on bool) error {
	if r.failSet {
		return errFake
	}
	if r.on != on {
		r.switches++
	}
	r.on = on
	return nil
}

func (r *fakeRelay) read(v float64) (float64, error) {
	if r.failGet {
		return 0, errFake
	}
	return v, nil
}

func (r *fakeRelay) Status() (api.ChargeStatus, error) { return api.StatusB, nil }
func (r *fakeRelay) MaxCurrent(int64) error            { return nil }
func (r *fakeRelay) CurrentPower() (float64, error)    { return r.read(r.power) }
func (r *fakeRelay) TotalEnergy() (float64, error)     { return r.read(r.energy) }
func (r *fakeRelay) Currents() (float64, float64, float64, error) {
	// e.g. a three-phase capable relay: the stage current is the sum
	i, err := r.read(r.amps)
	return i / 2, i / 2, 0, err
}

// switchOnly is a relay without measurement
type switchOnly struct{ r *fakeRelay }

func (s switchOnly) Status() (api.ChargeStatus, error) { return s.r.Status() }
func (s switchOnly) Enabled() (bool, error)            { return s.r.Enabled() }
func (s switchOnly) Enable(on bool) error              { return s.r.Enable(on) }
func (s switchOnly) MaxCurrent(i int64) error          { return s.r.MaxCurrent(i) }

func newTestHeater(t *testing.T, hysteresis float64, fakes ...*fakeRelay) *StagedHeater {
	t.Helper()

	relays := make([]api.Charger, len(fakes))
	for i, f := range fakes {
		relays[i] = f
	}

	wb, err := NewStagedHeater(&embed{}, relays, 1500, hysteresis)
	require.NoError(t, err)

	return wb
}

func threeRelays() []*fakeRelay {
	return []*fakeRelay{{}, {}, {}}
}

func withPhases(t *testing.T, wb *StagedHeater, phases int) {
	t.Helper()
	lp := loadpoint.NewMockAPI(gomock.NewController(t))
	lp.EXPECT().ActivePhases().Return(phases).AnyTimes()
	wb.LoadpointControl(lp)
}

func states(fakes []*fakeRelay) []bool {
	res := make([]bool, len(fakes))
	for i, f := range fakes {
		res[i] = f.on
	}
	return res
}

// current returns the per-phase current for the given total power
func current(power float64, phases int) float64 {
	return power / 230 / float64(phases)
}

func TestStagedHeaterStagesForPower(t *testing.T) {
	below := func(f float64) float64 { return f - 0.001 }

	for _, tc := range []struct {
		hysteresis, power float64
		active, want      int
	}{
		{0, 0, 0, 1}, // enabled heater keeps at least one stage
		{0, below(3000), 0, 1},
		{0, 3000, 0, 2},
		{0, math.Nextafter(3000, 0), 0, 2}, // float tolerance at boundaries
		{0, 10000, 0, 3},
		{0, below(3000), 3, 1}, // dropping is immediate
		{300, 3000, 0, 2},      // enabling: no hysteresis
		{300, below(3300), 1, 1},
		{300, 3300, 1, 2},
		{300, 6300, 1, 3},
		{300, below(4500), 2, 2},
		{300, 4500, 2, 3}, // last stage: no hysteresis, the loadpoint never requests more
		{300, below(3000), 2, 1},
	} {
		wb := newTestHeater(t, tc.hysteresis, threeRelays()...)
		assert.Equal(t, tc.want, wb.stagesForPower(tc.power, tc.active), "%+v", tc)
	}
}

func TestStagedHeaterConfig(t *testing.T) {
	relays := []api.Charger{new(fakeRelay)}

	for _, step := range []float64{0, -1, 499, math.NaN(), math.Inf(1)} {
		_, err := NewStagedHeater(&embed{}, relays, step, 0)
		assert.Error(t, err, "step %v", step)
	}

	for _, hyst := range []float64{-1, 1500, math.NaN()} {
		_, err := NewStagedHeater(&embed{}, relays, 1500, hyst)
		assert.Error(t, err, "hysteresis %v", hyst)
	}

	for _, n := range []int{0, 4} {
		_, err := NewStagedHeater(&embed{}, make([]api.Charger, n), 1500, 0)
		assert.Error(t, err, "%d stages", n)
	}

	wb, err := NewStagedHeater(&embed{}, relays, 1500, 0)
	require.NoError(t, err)
	for _, c := range []float64{-1, math.NaN(), math.Inf(1)} {
		assert.Error(t, wb.MaxCurrentMillis(c), "current %v", c)
	}
}

func TestStagedHeaterControl(t *testing.T) {
	for _, phases := range []int{1, 3} {
		fakes := threeRelays()
		wb := newTestHeater(t, 0, fakes...)
		withPhases(t, wb, phases)

		// loadpoint sets the current before enabling
		require.NoError(t, wb.MaxCurrentMillis(current(4500, phases)))
		assert.Equal(t, []bool{false, false, false}, states(fakes), "disabled heater must not switch")

		require.NoError(t, wb.Enable(true))
		assert.Equal(t, []bool{true, true, true}, states(fakes))

		for _, tc := range []struct {
			power float64
			want  []bool
		}{
			{3000, []bool{true, true, false}},
			{2999, []bool{true, false, false}},
			{4600, []bool{true, true, true}},
		} {
			require.NoError(t, wb.MaxCurrentMillis(current(tc.power, phases)))
			assert.Equal(t, tc.want, states(fakes), "%dp %v W", phases, tc.power)

			c, err := wb.GetMaxCurrent()
			require.NoError(t, err)
			assert.Equal(t, current(tc.power, phases), c, "requested current is reported")
		}

		require.NoError(t, wb.Enable(false))
		assert.Equal(t, []bool{false, false, false}, states(fakes))
	}
}

func TestStagedHeaterHysteresis(t *testing.T) {
	fakes := threeRelays()
	wb := newTestHeater(t, 300, fakes...)
	withPhases(t, wb, 3)

	require.NoError(t, wb.Enable(true))

	// power oscillating around the stage boundary must not switch relays
	for _, p := range []float64{2900, 3100, 2950, 3250} {
		require.NoError(t, wb.MaxCurrentMillis(current(p, 3)))
	}
	assert.Equal(t, 0, fakes[1].switches)

	require.NoError(t, wb.MaxCurrentMillis(current(3300, 3)))
	assert.Equal(t, []bool{true, true, false}, states(fakes))
	require.NoError(t, wb.MaxCurrentMillis(current(2999, 3)))
	assert.Equal(t, []bool{true, false, false}, states(fakes))
}

func TestStagedHeaterFailures(t *testing.T) {
	fakes := threeRelays()
	wb := newTestHeater(t, 0, fakes...)
	withPhases(t, wb, 3)
	require.NoError(t, wb.Enable(true))

	// switching on stops at the failing stage and is reported for a retry
	fakes[1].failSet = true
	assert.Error(t, wb.MaxCurrentMillis(current(4500, 3)))
	assert.Equal(t, []bool{true, false, false}, states(fakes))
	c, err := wb.GetMaxCurrent()
	require.NoError(t, err)
	assert.Zero(t, c)

	// switching off continues past the failing stage
	fakes[1].failSet = false
	require.NoError(t, wb.MaxCurrentMillis(current(4500, 3)))
	fakes[2].failSet = true
	assert.Error(t, wb.MaxCurrentMillis(current(1500, 3)))
	assert.Equal(t, []bool{true, false, true}, states(fakes))
	assert.Error(t, wb.Enable(false))
	assert.Equal(t, []bool{false, false, true}, states(fakes))

	// an unreachable relay keeps its last known state, so disabling is retried
	fakes[2].failGet = true
	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	// its readings are estimated from that state
	power, err := meterOf(wb).CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 1500.0, power)
	_, _, l3, err := currentsOf(wb).Currents()
	require.NoError(t, err)
	assert.InDelta(t, 1500.0/230, l3, 1e-9)

	status, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)

	// energy must be complete
	_, err = energyOf(wb).TotalEnergy()
	assert.Error(t, err)

	fakes[2].failGet, fakes[2].failSet = false, false
	require.NoError(t, wb.Enable(false))
	enabled, err = wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)

	// no relay reachable
	for _, f := range fakes {
		f.failGet = true
	}
	_, err = wb.Enabled()
	assert.Error(t, err)
	_, err = wb.Status()
	assert.Error(t, err)
	_, _, _, err = currentsOf(wb).Currents()
	assert.Error(t, err)
}

func TestStagedHeaterExternalState(t *testing.T) {
	fakes := threeRelays()
	wb := newTestHeater(t, 0, fakes...)
	withPhases(t, wb, 3)

	require.NoError(t, wb.MaxCurrentMillis(current(1500, 3)))
	require.NoError(t, wb.Enable(true))

	for _, manual := range [][]bool{{true, true, false}, {false, false, true}} {
		for i, on := range manual {
			fakes[i].on = on
		}
		_, err := wb.Enabled()
		require.NoError(t, err)

		// deviation is reported, so the loadpoint sets the current again
		c, err := wb.GetMaxCurrent()
		require.NoError(t, err)
		assert.Zero(t, c)

		require.NoError(t, wb.MaxCurrentMillis(current(1500, 3)))
		assert.Equal(t, []bool{true, false, false}, states(fakes))

		c, err = wb.GetMaxCurrent()
		require.NoError(t, err)
		assert.Equal(t, current(1500, 3), c)
	}

	// disabling switches off relays regardless of their last known state
	fakes[2].on = true
	require.NoError(t, wb.Enable(false))
	assert.Equal(t, []bool{false, false, false}, states(fakes))

	// a failing relay that was off is not assumed to be on
	require.NoError(t, wb.Enable(true))
	fakes[2].failSet, fakes[2].failGet = true, true
	assert.Error(t, wb.Enable(false))
	power, err := meterOf(wb).CurrentPower()
	require.NoError(t, err)
	assert.Zero(t, power)
}

func TestStagedHeaterStartup(t *testing.T) {
	fakes := []*fakeRelay{{on: true}, {on: true}, {on: true}}
	wb := newTestHeater(t, 300, fakes...)

	// the constructor adopts the running heater: with the third relay unreadable,
	// its power is estimated from the state read at construction
	fakes[2].failGet = true
	power, err := meterOf(wb).CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 1500.0, power)
	fakes[2].failGet = false

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled, "running heater is adopted")

	// loadpoint sets min current before LoadpointControl: never exceed the request
	minPower, _, err := wb.GetMinMaxPower()
	require.NoError(t, err)
	require.NoError(t, wb.MaxCurrentMillis(current(minPower, 3)))
	assert.Equal(t, []bool{true, false, false}, states(fakes))

	_, err = NewStagedHeater(&embed{}, []api.Charger{&fakeRelay{failGet: true}}, 1500, 0)
	assert.Error(t, err, "no known state for unreachable relay")
}

func TestStagedHeaterMeasurement(t *testing.T) {
	fakes := []*fakeRelay{
		{power: 1490, energy: 1, amps: 6.5},
		{energy: 2},
		{power: 1510, energy: 3.5, amps: 6.6},
	}
	wb := newTestHeater(t, 0, fakes...)

	minP, maxP, err := wb.GetMinMaxPower()
	require.NoError(t, err)
	assert.Equal(t, []float64{1500, 4500}, []float64{minP, maxP})

	power, err := meterOf(wb).CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 3000.0, power)

	energy, err := energyOf(wb).TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 6.5, energy)

	// stage 1-3 is phase L1-L3
	l1, l2, l3, err := currentsOf(wb).Currents()
	require.NoError(t, err)
	assert.Equal(t, []float64{6.5, 0, 6.6}, []float64{l1, l2, l3})

	// status follows the measured power: relay on with open thermostat is not heating
	status, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)
	fakes[0].on, fakes[0].power, fakes[2].power = true, 0, 50
	status, err = wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusB, status)
}

func TestStagedHeaterSwitchOnly(t *testing.T) {
	fakes := threeRelays()
	relays := []api.Charger{switchOnly{fakes[0]}, switchOnly{fakes[1]}, fakes[2]}

	wb, err := NewStagedHeater(&embed{}, relays, 1500, 0)
	require.NoError(t, err)

	// measurements are only provided if all relays measure
	_, ok := api.Cap[api.Meter](wb)
	assert.False(t, ok)
	_, ok = api.Cap[api.MeterEnergy](wb)
	assert.False(t, ok)
	_, ok = api.Cap[api.PhaseCurrents](wb)
	assert.False(t, ok)

	// status follows the relays
	status, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusB, status)

	require.NoError(t, wb.Enable(true))
	status, err = wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)
}

func TestStagedHeaterFromConfig(t *testing.T) {
	ctx := t.Context()
	fakes := threeRelays()

	config.Reset()
	t.Cleanup(config.Reset)
	for i, f := range fakes[:2] {
		require.NoError(t, config.Chargers().Add(config.NewStaticDevice[api.Charger](config.Named{Name: fmt.Sprintf("relay%d", i+1)}, f)))
	}
	registry.AddCtx("test-stage", func(context.Context, map[string]any) (api.Charger, error) {
		return fakes[2], nil
	})

	device := func(name string) map[string]any { return map[string]any{"type": "device", "name": name} }

	c, err := NewStagedHeaterFromConfig(ctx, map[string]any{
		"stage1": device("relay1"), "stage2": device("relay2"), "stage3": map[string]any{"type": "test-stage"},
		"steppower": 1500,
	})
	require.NoError(t, err)

	wb := c.(*StagedHeater)
	assert.Equal(t, 300.0, wb.hysteresis, "default hysteresis")

	require.NoError(t, wb.MaxCurrentMillis(current(4500, 1)))
	require.NoError(t, wb.Enable(true))
	assert.Equal(t, []bool{true, true, true}, states(fakes))

	for _, stages := range []map[string]any{
		{},
		{"stage1": device("relay1"), "stage2": device("relay1")},                                   // duplicate
		{"stage1": device("relay1"), "stage2": map[string]any{"type": "DEVICE", "name": "relay1"}}, // duplicate
		{"stage1": device("relay1"), "stage3": device("relay2")},                                   // gap
		{"stage1": device("relay1"), "stage2": map[string]any{"type": "device"}},                   // missing name
		{"stage1": device("missing")},
	} {
		stages["steppower"] = 1500
		_, err := NewStagedHeaterFromConfig(ctx, stages)
		assert.Error(t, err, "stages %v", stages)
	}

	// template: stage 3 without stage 2 must not shift to phase 2
	_, err = NewFromConfig(ctx, "template", map[string]any{
		"template": "staged-heater", "stage1": "relay1", "stage3": "relay2", "steppower": 1500,
	})
	assert.ErrorContains(t, err, "stage 2: missing")
}

func meterOf(c any) api.Meter {
	m, _ := api.Cap[api.Meter](c)
	return m
}

func energyOf(c any) api.MeterEnergy {
	m, _ := api.Cap[api.MeterEnergy](c)
	return m
}

func currentsOf(c any) api.PhaseCurrents {
	m, _ := api.Cap[api.PhaseCurrents](c)
	return m
}

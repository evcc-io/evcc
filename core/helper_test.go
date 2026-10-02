package core

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrentToPowerGridType(t *testing.T) {
	Voltage = 230
	t.Cleanup(func() { GridIT = false })

	for _, tc := range []struct {
		it     bool
		phases int
		power  float64
	}{
		{false, 1, 2300},
		{false, 3, 6900},
		{true, 1, 2300},
		{true, 3, 3983.71},
	} {
		GridIT = tc.it

		power := currentToPower(10, tc.phases)
		assert.InDelta(t, tc.power, power, 0.01, "it=%v phases=%d", tc.it, tc.phases)
		assert.InDelta(t, 10, powerToCurrent(power, tc.phases), 1e-9, "it=%v phases=%d", tc.it, tc.phases)
	}
}

func TestEffectivePowerGridIT(t *testing.T) {
	Voltage = 230
	GridIT = true
	t.Cleanup(func() { GridIT = false })

	lp := &Loadpoint{
		phasesConfigured: 3,
		phases:           3,
		minCurrent:       6,
		maxCurrent:       16,
	}

	assert.InDelta(t, 2390.23, lp.EffectiveMinPower(), 0.01)
	assert.InDelta(t, 6373.94, lp.EffectiveMaxPower(), 0.01)
	assert.InDelta(t, 398.37, lp.EffectiveStepPower(), 0.01)
}

func TestPhasesFromChargeCurrentsGridIT(t *testing.T) {
	t.Cleanup(func() { GridIT = false })

	for _, tc := range []struct {
		it       bool
		currents []float64
		phases   int
	}{
		{false, []float64{10, 10, 0}, 2},
		{true, []float64{10, 10, 0}, 1},
		{true, []float64{0, 10, 10}, 1},
		{true, []float64{10, 10, 10}, 3},
	} {
		GridIT = tc.it

		lp := &Loadpoint{
			log:            util.NewLogger("foo"),
			clock:          clock.NewMock(),
			status:         api.StatusC,
			chargeCurrents: tc.currents,
		}

		lp.phasesFromChargeCurrents()
		require.Equal(t, tc.phases, lp.GetMeasuredPhases(), "it=%v currents=%v", tc.it, tc.currents)
	}
}

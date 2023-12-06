package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemainingPlanEnergy(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), nil)

	until := time.Now().Add(10 * time.Hour)
	energy := 10.0

	require.NoError(t, lp.SetPlanEnergy(until, energy))

	lp.sessionEnergy.Update(2 * energy)

	e, ok := lp.remainingPlanEnergy()
	assert.Equal(t, e, 0.0)
	assert.True(t, ok)
}

func TestRemainingPlanSoc(t *testing.T) {
}

func TestPlanRequiredDuration(t *testing.T) {
	Voltage = 230
	clock := clock.NewMock()

	tariff, _ := tariff.NewFixed(0.3, nil, tariff.FixedWithClock(clock))
	// tariff = nil

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.clock = clock
	lp.planner = planner.New(lp.log, tariff, planner.WithClock(clock))

	const (
		power  = 3680.0          // W
		energy = 2 * power / 1e3 // kWh
		phases = 1               // 1p
	)

	until := clock.Now().Add(10 * time.Hour)
	require.NoError(t, lp.SetPlanEnergy(until, energy))

	e, ok := lp.remainingPlanEnergy()
	assert.True(t, ok)
	assert.Equal(t, energy, e)

	ts := lp.EffectivePlanTime()
	assert.Equal(t, until, ts)

	lp.phases = phases
	maxPower := lp.EffectiveMaxPower()
	assert.Equal(t, Voltage*lp.MaxCurrent*phases, maxPower)

	d := lp.planRequiredDuration(maxPower)
	assert.Equal(t, 2*time.Hour, d)

	plan, err := lp.planner.Plan(d, until)
	require.NoError(t, err)
	require.NotEmpty(t, plan)

	assert.False(t, lp.plannerActive())

	clock.Add(8 * time.Hour)
	assert.Equal(t, 2*time.Hour, lp.planRequiredDuration(maxPower))
	assert.True(t, lp.plannerActive())

	clock.Add(2 * time.Hour)
	lp.sessionEnergy.Update(2 * energy)

	assert.Equal(t, time.Duration(0), lp.planRequiredDuration(maxPower))
	assert.False(t, lp.plannerActive())
}

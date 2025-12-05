package planner

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestContinuous_SinglePlanWindow(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)
	ctrl := gomock.NewController(t)

	rates := api.Rates{
		{Start: now, End: now.Add(1 * time.Hour), Value: 0.09},
		{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour), Value: 0.20},
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.10},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.11},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.11},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.25},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  c,
		tariff: trf,
	}

	plan := p.Plan(2*time.Hour, 0, now.Add(6*time.Hour), true)

	require.Len(t, plan, 2)
	assert.Equal(t, rates[2].Start, plan[0].Start)
	assert.Equal(t, rates[3].End, plan[len(plan)-1].End)
	assert.Equal(t, rates[2].Value, plan[0].Value)
	assert.Equal(t, rates[3].Value, plan[1].Value)
}

func TestContinuous_WindowWithPastRates(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)

	rates := api.Rates{
		{Start: now.Add(-3 * time.Hour), End: now.Add(-2 * time.Hour), Value: 0.05},
		{Start: now.Add(-2 * time.Hour), End: now.Add(-1 * time.Hour), Value: 0.08},
		{Start: now.Add(-1 * time.Hour), End: now, Value: 0.07},
		{Start: now, End: now.Add(1 * time.Hour), Value: 0.20},
		{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour), Value: 0.09},
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.10},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.15},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.11},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.25},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  c,
		tariff: trf,
	}

	targetTime := now.Add(6 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := p.Plan(requiredDuration, 0, targetTime, true)

	require.NotEmpty(t, plan)
	require.Len(t, plan, 2)
	assert.False(t, plan[0].Start.Before(now))
	assert.Equal(t, now.Add(1*time.Hour), plan[0].Start)
	assert.Equal(t, now.Add(3*time.Hour), plan[len(plan)-1].End)
	assert.Equal(t, 0.09, plan[0].Value)
	assert.Equal(t, 0.10, plan[1].Value)
}

func TestContinuous_WindowAllRatesInPast(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)

	rates := api.Rates{
		{Start: now.Add(-6 * time.Hour), End: now.Add(-5 * time.Hour), Value: 0.05},
		{Start: now.Add(-5 * time.Hour), End: now.Add(-4 * time.Hour), Value: 0.08},
		{Start: now.Add(-4 * time.Hour), End: now.Add(-3 * time.Hour), Value: 0.07},
		{Start: now.Add(-3 * time.Hour), End: now.Add(-2 * time.Hour), Value: 0.09},
		{Start: now.Add(-2 * time.Hour), End: now.Add(-1 * time.Hour), Value: 0.10},
		{Start: now.Add(-1 * time.Hour), End: now, Value: 0.11},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  c,
		tariff: trf,
	}

	targetTime := now.Add(3 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := p.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	// When all rates are in the past and target is in future, expect nil plan
	assert.Empty(t, plan, "plan should be nil when all rates are in the past")
}

// TestContinuous_WindowRatesSpanningPastAndFuture tests continuous mode with rates
// spanning from past to future, where the optimal window would start in the past
func TestContinuous_WindowRatesSpanningPastAndFuture(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)

	// Rates spanning from 3h before now to 6h after now
	// The cheapest window would be -3h to -1h, but that's in the past
	rates := api.Rates{
		{Start: now.Add(-3 * time.Hour), End: now.Add(-2 * time.Hour), Value: 0.05}, // cheapest, but past
		{Start: now.Add(-2 * time.Hour), End: now.Add(-1 * time.Hour), Value: 0.06}, // cheap, but past
		{Start: now.Add(-1 * time.Hour), End: now, Value: 0.12},                     // partially past
		{Start: now, End: now.Add(1 * time.Hour), Value: 0.15},
		{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour), Value: 0.08}, // cheapest future
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.09}, // second cheapest future
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.18},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.14},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.20},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  c,
		tariff: trf,
	}

	targetTime := now.Add(6 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := p.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	require.NotEmpty(t, plan)
	require.Len(t, plan, 2)

	// Critical: plan must start at or after now, even if cheaper rates existed in the past
	assert.False(t, plan[0].Start.Before(now), "plan must not start in the past")

	// Should find cheapest 2-hour window starting from now or later
	// Expected: 1h-3h window (two slots with prices 0.08 and 0.09)
	assert.Equal(t, now.Add(1*time.Hour), plan[0].Start, "start should be at cheapest future window")
	assert.Equal(t, now.Add(3*time.Hour), plan[len(plan)-1].End, "end should match 2-hour window")
	assert.Equal(t, 0.08, plan[0].Value, "first slot should have actual price")
	assert.Equal(t, 0.09, plan[1].Value, "second slot should have actual price")
}

// TestContinuous_WindowRatesStartInFuture tests continuous mode when tariff data
// starts in the future, but target time is within the tariff data range
func TestContinuous_WindowRatesStartInFuture(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)

	rates := api.Rates{
		{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour), Value: 0.20},
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.08},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.09},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.15},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.18},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  c,
		tariff: trf,
	}

	targetTime := now.Add(5 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := p.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	require.NotEmpty(t, plan)
	require.Len(t, plan, 2)

	// Plan must not start in the past
	assert.False(t, plan[0].Start.Before(now), "plan must not start in the past")

	// Should find cheapest 2-hour window within available rates
	// Expected: 2h-4h window (two slots with prices 0.08 and 0.09)
	assert.Equal(t, now.Add(2*time.Hour), plan[0].Start, "start should be at cheapest window in future rates")
	assert.Equal(t, now.Add(4*time.Hour), plan[len(plan)-1].End, "end should match 2-hour window")
	assert.Equal(t, 0.08, plan[0].Value, "first slot should have actual price")
	assert.Equal(t, 0.09, plan[1].Value, "second slot should have actual price")
}

func TestContinuous_WindowLateChargingPreference(t *testing.T) {
	now := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	ctrl := gomock.NewController(t)

	rates := api.Rates{
		{Start: now, End: now.Add(1 * time.Hour), Value: 0.10},
		{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour), Value: 0.10},
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.10},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.10},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.10},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.15},
	}

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates, nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  c,
		tariff: trf,
	}

	targetTime := now.Add(6 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := p.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	require.NotEmpty(t, plan)
	require.Len(t, plan, 2)

	// Should select the latest window with equal cost (3h-5h)
	// All windows from 0h-2h, 1h-3h, 2h-4h, and 3h-5h have the same total cost
	// But we prefer late charging, so 3h-5h should be selected
	assert.Equal(t, now.Add(3*time.Hour), plan[0].Start, "should select latest window with equal cost")
	assert.Equal(t, now.Add(5*time.Hour), plan[len(plan)-1].End, "end should be 2 hours after start")
	assert.Equal(t, 0.10, plan[0].Value, "first slot should have actual price")
	assert.Equal(t, 0.10, plan[1].Value, "second slot should have actual price")
}

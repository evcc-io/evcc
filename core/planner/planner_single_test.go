package planner

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTariff implements api.Tariff for testing single-plan mode
type MockTariff struct {
	rates api.Rates
}

func (m *MockTariff) Rates() (api.Rates, error) {
	return m.rates, nil
}

// Return a valid api.TariffType for compilation (dummy value for tests)
func (m *MockTariff) Type() api.TariffType {
	return 0
}

// TestSinglePlanContinuousWindow tests the single continuous cheapest window mode
func TestSinglePlanContinuousWindow(t *testing.T) {
	now := time.Date(2025, 10, 15, 0, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	log := util.NewLogger("test")

	rates := api.Rates{
		{Start: now, End: now.Add(1 * time.Hour), Value: 0.09},
		{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour), Value: 0.20},
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.10},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.11},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.11},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.25},
	}

	tariff := &MockTariff{rates: rates}
	planner := &Planner{
		log:    log,
		clock:  c,
		tariff: tariff,
	}

	targetTime := now.Add(6 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := planner.Plan(requiredDuration, 0, targetTime, true) // single continuous mode

	require.Len(t, plan, 1, "should create a single continuous slot")
	assert.Equal(t, rates[2].Start, plan[0].Start, "start of the plan should match cheapest slot")
	assert.Equal(t, rates[3].End, plan[0].End, "end of the plan should match cheapest slot")

	const delta = 0.01
	assert.InDelta(t, 0.105, plan[0].Value, delta, "plan value should be the cheapest")
}

// TestContinuousWindowWithPastRates tests that plans in continuous mode
// do not start in the past when some tariff data is already outdated
func TestContinuousWindowWithPastRates(t *testing.T) {
	now := time.Date(2025, 10, 15, 12, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	log := util.NewLogger("test")

	// Create rates that include past data (3h before now, 6h after now)
	rates := api.Rates{
		// Past rates (should be ignored)
		{Start: now.Add(-3 * time.Hour), End: now.Add(-2 * time.Hour), Value: 0.05}, // cheapest, but in the past
		{Start: now.Add(-2 * time.Hour), End: now.Add(-1 * time.Hour), Value: 0.08},
		{Start: now.Add(-1 * time.Hour), End: now, Value: 0.07},
		// Future rates
		{Start: now, End: now.Add(1 * time.Hour), Value: 0.20},
		{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour), Value: 0.09}, // cheapest future slot
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.10},
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.15},
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.11},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.25},
	}

	tariff := &MockTariff{rates: rates}
	planner := &Planner{
		log:    log,
		clock:  c,
		tariff: tariff,
	}

	targetTime := now.Add(6 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := planner.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	require.NotEmpty(t, plan, "plan should not be empty")
	require.Len(t, plan, 1, "should create a single continuous slot")

	// Critical assertion: plan must not start in the past
	assert.False(t, plan[0].Start.Before(now), "plan must not start in the past")
	assert.GreaterOrEqual(t, plan[0].Start.Unix(), now.Unix(), "plan start must be >= now")

	// Plan should find the cheapest 2-hour window in the future
	// Expected: 1h-3h (avg price = (0.09 + 0.10) / 2 = 0.095)
	assert.Equal(t, now.Add(1*time.Hour), plan[0].Start, "start should be at the cheapest future window")
	assert.Equal(t, now.Add(3*time.Hour), plan[0].End, "end should match 2-hour window")

	const delta = 0.01
	expectedAvgPrice := (0.09 + 0.10) / 2
	assert.InDelta(t, expectedAvgPrice, plan[0].Value, delta, "plan value should be weighted average of cheapest window")
}

// TestContinuousWindowAllRatesInPast tests the edge case where all rates are in the past
func TestContinuousWindowAllRatesInPast(t *testing.T) {
	now := time.Date(2025, 10, 15, 12, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	log := util.NewLogger("test")

	// All rates are in the past
	rates := api.Rates{
		{Start: now.Add(-6 * time.Hour), End: now.Add(-5 * time.Hour), Value: 0.05},
		{Start: now.Add(-5 * time.Hour), End: now.Add(-4 * time.Hour), Value: 0.08},
		{Start: now.Add(-4 * time.Hour), End: now.Add(-3 * time.Hour), Value: 0.07},
		{Start: now.Add(-3 * time.Hour), End: now.Add(-2 * time.Hour), Value: 0.09},
		{Start: now.Add(-2 * time.Hour), End: now.Add(-1 * time.Hour), Value: 0.10},
		{Start: now.Add(-1 * time.Hour), End: now, Value: 0.11},
	}

	tariff := &MockTariff{rates: rates}
	planner := &Planner{
		log:    log,
		clock:  c,
		tariff: tariff,
	}

	targetTime := now.Add(3 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := planner.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	// When all rates are in the past and target is in future, expect nil plan
	assert.Empty(t, plan, "plan should be nil when all rates are in the past")
}

// TestContinuousWindowRatesSpanningPastAndFuture tests continuous mode with rates
// spanning from past to future, where the optimal window would start in the past
func TestContinuousWindowRatesSpanningPastAndFuture(t *testing.T) {
	now := time.Date(2025, 10, 15, 12, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	log := util.NewLogger("test")

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

	tariff := &MockTariff{rates: rates}
	planner := &Planner{
		log:    log,
		clock:  c,
		tariff: tariff,
	}

	targetTime := now.Add(6 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := planner.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	require.NotEmpty(t, plan, "plan should not be empty")
	require.Len(t, plan, 1, "should create a single continuous slot")

	// Critical: plan must start at or after now, even if cheaper rates existed in the past
	assert.False(t, plan[0].Start.Before(now), "plan must not start in the past")
	assert.GreaterOrEqual(t, plan[0].Start.Unix(), now.Unix(), "plan start must be >= now")

	// Should find cheapest 2-hour window starting from now or later
	// Expected: 1h-3h window (0.08 + 0.09) / 2 = 0.085
	assert.Equal(t, now.Add(1*time.Hour), plan[0].Start, "start should be at cheapest future window")
	assert.Equal(t, now.Add(3*time.Hour), plan[0].End, "end should match 2-hour window")

	const delta = 0.01
	expectedAvgPrice := (0.08 + 0.09) / 2
	assert.InDelta(t, expectedAvgPrice, plan[0].Value, delta, "plan value should be weighted average")
}

// TestContinuousWindowRatesStartInFuture tests continuous mode when tariff data
// starts in the future, but target time is within the tariff data range
func TestContinuousWindowRatesStartInFuture(t *testing.T) {
	now := time.Date(2025, 10, 15, 12, 0, 0, 0, time.UTC)
	c := clock.NewMock()
	c.Set(now)

	log := util.NewLogger("test")

	// Rates start 1 hour in the future, no rates available for now
	rates := api.Rates{
		{Start: now.Add(1 * time.Hour), End: now.Add(2 * time.Hour), Value: 0.20},
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 0.08}, // cheapest
		{Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Value: 0.09}, // second cheapest
		{Start: now.Add(4 * time.Hour), End: now.Add(5 * time.Hour), Value: 0.15},
		{Start: now.Add(5 * time.Hour), End: now.Add(6 * time.Hour), Value: 0.18},
	}

	tariff := &MockTariff{rates: rates}
	planner := &Planner{
		log:    log,
		clock:  c,
		tariff: tariff,
	}

	targetTime := now.Add(5 * time.Hour)
	requiredDuration := 2 * time.Hour

	plan := planner.Plan(requiredDuration, 0, targetTime, true) // continuous mode

	require.NotEmpty(t, plan, "plan should not be empty")
	require.Len(t, plan, 1, "should create a single continuous slot")

	// Plan must not start in the past
	assert.False(t, plan[0].Start.Before(now), "plan must not start in the past")
	assert.GreaterOrEqual(t, plan[0].Start.Unix(), now.Unix(), "plan start must be >= now")

	// Should find cheapest 2-hour window within available rates
	// Expected: 2h-4h window (0.08 + 0.09) / 2 = 0.085
	assert.Equal(t, now.Add(2*time.Hour), plan[0].Start, "start should be at cheapest window in future rates")
	assert.Equal(t, now.Add(4*time.Hour), plan[0].End, "end should match 2-hour window")

	const delta = 0.01
	expectedAvgPrice := (0.08 + 0.09) / 2
	assert.InDelta(t, expectedAvgPrice, plan[0].Value, delta, "plan value should be weighted average")
}

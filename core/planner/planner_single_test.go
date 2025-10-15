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

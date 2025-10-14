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

// TestPlannerMultipleWindows verifies that Planner.plan generates multiple windows when necessary
func TestPlannerMultipleWindows(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0, 1, 2, 3, 1, 5, 6, 7}, clock.Now(), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	// Plan a charging duration spanning multiple slots
	plan0 := p.Plan(2*time.Hour, 0, clock.Now().Add(8*time.Hour))
	require.NotNil(t, plan0)

	plan0p := p.Plan(2*time.Hour, 15*time.Minute, clock.Now().Add(8*time.Hour))
	require.NotNil(t, plan0)

	// Single window optimization (minGap=0)
	plan1 := p.Plan(2*time.Hour, 0, clock.Now().Add(8*time.Hour), 0)
	require.NotNil(t, plan1)

	plan1p := p.Plan(2*time.Hour, 15*time.Minute, clock.Now().Add(8*time.Hour), 0)
	require.NotNil(t, plan1p)

	// Minimize charging windows (minGap>1)
	plan2 := p.Plan(2*time.Hour, 0, clock.Now().Add(8*time.Hour), 0)
	require.NotNil(t, plan2)

	// Count the number of charging windows
	windows0 := countChargingWindows(plan0)
	windows0p := countChargingWindows(plan0p)
	windows1 := countChargingWindows(plan1)
	windows1p := countChargingWindows(plan1p)
	windows2 := countChargingWindows(plan2)
	
	t.Logf("Number of charging windows (cost-optimized): %d", windows0)
	t.Logf("Number of charging windows (cost-optimized+precondition): %d", windows0p)
	t.Logf("Number of charging windows (single): %d", windows1)
	t.Logf("Number of charging windows (single+precondition): %d", windows1p)
	t.Logf("Number of charging windows (minimize windows): %d", windows2)

	// We expect at least 2 windows due to gaps in the tariff
	assert.GreaterOrEqual(t, 2, windows0, "expected more than one charging window")
	assert.Equal(t, windows0+1, windows0p, "expected one additional window for precondition")
	assert.Equal(t, 1, windows1, "expected a single charging window")
	assert.Equal(t, 2, windows1p, "expected single window plus precondition window")
	assert.GreaterOrEqual(t, windows0, windows2, "expected minimized window count")
}
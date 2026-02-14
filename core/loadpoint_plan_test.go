package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func setupTestLoadpoint(t *testing.T, mockClock *clock.Mock) (*Loadpoint, *gomock.Controller) {
	ctrl := gomock.NewController(t)

	Voltage = 230 // V

	lp := NewLoadpoint(util.NewLogger("test"), nil)
	lp.clock = mockClock

	// Setup push channel to prevent blocking
	lp.pushChan = make(chan push.Event, 10)

	// Setup minimal required mocks
	charger := api.NewMockCharger(ctrl)

	lp.charger = charger
	lp.chargeMeter = &Null{}
	lp.chargeRater = &Null{}
	lp.chargeTimer = &Null{}

	// Mock connected state
	charger.EXPECT().Status().Return(api.StatusC, nil).AnyTimes()
	lp.status = api.StatusC

	// Setup charging parameters (typical 3-phase 16A setup = 11kW)
	// Real logs show 7.36kW, but test uses 11kW for consistency
	lp.minCurrent = 6
	lp.maxCurrent = 16
	lp.phases = 3
	lp.phasesConfigured = 3

	return lp, ctrl
}

// TestPlannerActive_StopWhenOnlyPreconditionRemains_ShortDuration tests
// avoid: "plan: continuing for remaining 11m0s" when requiredDuration < precondition
//
// scenario:
// - Time: 02:30:38
// - Target: 07:00:00 (4.5 hours away)
// - Required: 11m
// - Precondition: 15 minutes (900 seconds)
// - 660s < 900s = precondition-only situation
// - Should: Stop immediately, charge later closer to target
func TestPlannerActive_StopWhenOnlyPreconditionRemains_ShortDuration(t *testing.T) {
	mockClk := clock.NewMock()
	// Simulate 02:30:38
	now := time.Date(2026, 2, 4, 2, 30, 38, 0, time.Local)
	mockClk.Set(now)

	lp, ctrl := setupTestLoadpoint(t, mockClk)
	defer ctrl.Finish()

	// Target time: 07:00:00 (4h 29m 22s away)
	targetTime := time.Date(2026, 2, 4, 7, 0, 0, 0, time.Local)

	// Uniform tariff (cost doesn't matter for this test)
	tariffRates := api.Rates{
		{Start: now.Add(-time.Hour), End: now.Add(5 * time.Hour), Value: 0.268},
	}
	mockTariff := api.NewMockTariff(ctrl)
	mockTariff.EXPECT().Rates().Return(tariffRates, nil).AnyTimes()
	lp.planner = planner.New(lp.log, mockTariff)

	// State:
	// - planActive = true (was charging)
	// - requiredDuration = 11m (via planEnergy calculation)
	// - precondition = 15 minutes (default)
	lp.planActive = true
	lp.planSlotEnd = time.Time{} // Not in a slot currently
	lp.planTime = targetTime
	lp.planEnergy = 2.68 // 11m at 7.36kW
	lp.planStrategy = api.PlanStrategy{
		Precondition: 15 * time.Minute, // Standard precondition
	}

	// Execute
	active := lp.plannerActive()

	// Verify: Should STOP because requiredDuration (660s) < precondition (900s)
	// returns false (stops, will charge later near target)
	assert.False(t, active, "Should stop when requiredDuration (11min) < precondition (15min)")
}

// TestPlannerActive_StopWhenOnlyPreconditionRemains_WithinSlot tests
// Variant of the above scenario but with planSlotEnd set (still within active slot)
// avoid: "plan: continuing until end of slot" when requiredDuration < precondition
//
// Scenario: Still charging in a slot that ends soon, but only precondition time remains
// - NOW: within active slot (slot ends in 10 minutes)
// - Required: 11m
// - Precondition: 15 minutes (900 seconds)
// - 660s < 900s = precondition-only situation
// - Should: Stop now, charge later closer to target
func TestPlannerActive_StopWhenOnlyPreconditionRemains_WithinSlot(t *testing.T) {
	mockClk := clock.NewMock()
	// Simulate within an active slot
	now := time.Date(2026, 2, 4, 2, 30, 38, 0, time.Local)
	mockClk.Set(now)

	lp, ctrl := setupTestLoadpoint(t, mockClk)
	defer ctrl.Finish()

	// Target time: 07:00:00 (4h 29m 22s away)
	targetTime := time.Date(2026, 2, 4, 7, 0, 0, 0, time.Local)

	// Uniform tariff (cost doesn't matter for this test)
	tariffRates := api.Rates{
		{Start: now.Add(-time.Hour), End: now.Add(5 * time.Hour), Value: 0.268},
	}
	mockTariff := api.NewMockTariff(ctrl)
	mockTariff.EXPECT().Rates().Return(tariffRates, nil).AnyTimes()
	lp.planner = planner.New(lp.log, mockTariff)

	// State: currently within an active slot (ends in 10 minutes)
	lp.planActive = true
	lp.planSlotEnd = now.Add(10 * time.Minute) // Still in slot, ends soon
	lp.planTime = targetTime
	lp.planEnergy = 2.68 // 11m at 7.36kW
	lp.planStrategy = api.PlanStrategy{
		Precondition: 15 * time.Minute, // Standard precondition
	}

	// Execute
	active := lp.plannerActive()

	// Verify: Should STOP because requiredDuration (660s) < precondition (900s)
	// Even though we're within a slot, we should stop if only precondition remains
	// returns false (stops, will charge later near target)
	assert.False(t, active, "Should stop even within slot when requiredDuration (11min) < precondition (15min)")
}

// TestPlannerActive_AvoidRestartWithinThreshold tests
// Scenario: Just finished charging in cheap slot, next is expensive (15min), then cheap again
// Threshold 14min allows stopping to skip the expensive slot
//
// Pattern (15-minute slots, NOW is 1s after slot boundary, depends on cycle time)):
// - Slot 0 (09:45-10:00): cheap 0.20, was charging here (just ended)
// - Slot 1 (10:00-10:15): expensive 0.50 (should SKIP)
// - Slot 2 (10:15-10:30): cheap 0.20 (resume here)
// - At NOW (10:00:01), next cheap planStart is 10:15 → 14m59s away
// - stops, skips expensive, resumes at cheap
func TestPlannerActive_AvoidRestartWithinThreshold(t *testing.T) {
	mockClk := clock.NewMock()
	// 1 second past slot boundary so planStart (10:15) is 14m59s away
	now := time.Date(2026, 2, 4, 10, 0, 1, 0, time.Local)
	mockClk.Set(now)

	lp, ctrl := setupTestLoadpoint(t, mockClk)
	defer ctrl.Finish()

	targetTime := now.Add(3 * time.Hour) // 3 hours to target

	// Clean 15-minute tariff slots aligned to 09:45, 10:00, 10:15, ...
	// Slot 2 is CHEAPEST, forcing planner to use it
	base := time.Date(2026, 2, 4, 9, 45, 0, 0, time.Local)
	slotPrices := []float64{
		0.25, // Slot 0: 09:45-10:00 (was charging here, ENDED)
		0.50, // Slot 1: 10:00-10:15 (EXPENSIVE, should skip!)
		0.20, // Slot 2: 10:15-10:30 (CHEAPEST! planner must use)
		0.50, // Slot 3-10: expensive
		0.50,
		0.50,
		0.50,
		0.50,
		0.50,
		0.50,
		0.50,
		0.30, // Slot 11: cheaper than expensive but more than slot 2
	}

	slotDuration := 15 * time.Minute
	tariffRates := make(api.Rates, len(slotPrices))
	for i, price := range slotPrices {
		tariffRates[i] = api.Rate{
			Start: base.Add(time.Duration(i) * slotDuration),
			End:   base.Add(time.Duration(i+1) * slotDuration),
			Value: price,
		}
	}

	mockTariff := api.NewMockTariff(ctrl)
	mockTariff.EXPECT().Rates().Return(tariffRates, nil).AnyTimes()

	lp.planner = planner.New(lp.log, mockTariff, planner.WithClock(mockClk))

	// State: was charging in slot that just ended
	lp.planActive = true
	lp.planSlotEnd = base.Add(slotDuration) // Slot 0 ended at 10:00
	lp.planTime = targetTime
	lp.planEnergy = 5.52 // 30min at 11kW
	lp.planStrategy = api.PlanStrategy{
		Continuous:   false,
		Precondition: 15 * time.Minute,
	}

	// Execute
	active := lp.plannerActive()

	// Verify: Should STOP (skip expensive slot 1, wait for cheap slot 2)
	// next cheap at 15min slot boundary (14m59s away) >= 14min → false (stop)
	assert.False(t, active, "Should stop to skip expensive slot when next cheap on next slot boundary")
}

// TestPlannerActive_ContinuousModeKeepsCharging tests continuous execution
// Validates that continuous mode prevents interruptions through expensive gap
//
// scenario:
// - 04:00-04:15: Charged in cheap slot (0.311 EUR/kWh)
// - 04:17: NOW - just after cheap slot ended (in expensive gap)
// - 04:15-05:00: Expensive gap (45 minutes, THREE 15-min expensive slots)
// - 05:00-07:00: Next cheap slot
// - Target: 07:00, Required: ~1h30m
// - Planner prefers: wait until 05:00 (at equal or unsignificant cheaper costs)
// - Continuous mode: keep charging continuous
func TestPlannerActive_ContinuousModeKeepsCharging(t *testing.T) {
	mockClk := clock.NewMock()
	// Simulate 04:17 (2 minutes after cheap slot ended)
	now := time.Date(2024, 1, 31, 4, 17, 0, 0, time.Local)
	mockClk.Set(now)

	lp, ctrl := setupTestLoadpoint(t, mockClk)
	defer ctrl.Finish()

	// Tariff structure: 15-minute slots
	// Previous cheap slot: 04:00-04:15 (ended 2 minutes ago)
	// constructed tariff gap: 04:15-05:00 (45 minutes, THREE 15-min more expensive slots)
	// not testing planner logic here, just that continuous mode keeps charging on alternative later plans (at usually equal costs)!
	// Next cheap: 05:00-07:00
	tariffRates := api.Rates{
		// Previous cheap slot (ended at 04:15)
		{Start: now.Add(-17 * time.Minute), End: now.Add(-2 * time.Minute), Value: 0.30},
		// Expensive slot 1: 04:15-04:30 (NOW is here at 04:17)
		{Start: now.Add(-2 * time.Minute), End: now.Add(13 * time.Minute), Value: 0.45},
		// Expensive slot 2: 04:30-04:45
		{Start: now.Add(13 * time.Minute), End: now.Add(28 * time.Minute), Value: 0.45},
		// Expensive slot 3: 04:45-05:00
		{Start: now.Add(28 * time.Minute), End: now.Add(43 * time.Minute), Value: 0.45},
		// Next cheap slot: 05:00-07:00
		{Start: now.Add(43 * time.Minute), End: now.Add(2*time.Hour + 43*time.Minute), Value: 0.30},
	}
	mockTariff := api.NewMockTariff(ctrl)
	mockTariff.EXPECT().Rates().Return(tariffRates, nil).AnyTimes()
	lp.planner = planner.New(lp.log, mockTariff, planner.WithClock(mockClk))

	// State matches scenario:
	// - Was charging 04:00-04:15, now at 04:17 (in more expensive gap)
	// - Still need ~1h30m of charging
	// - Target time: 07:00 (2h43m from now)
	// - Three constructed expensive slots ahead (45 min total gap)
	lp.planActive = true                                // Was charging in 04:00-04:15
	lp.planSlotEnd = time.Time{}                        // Not in active slot (gap between slots)
	lp.planTime = now.Add(2*time.Hour + 43*time.Minute) // Target: 07:00
	lp.planEnergy = 16.56                               // ~1h30m at 11kW
	lp.planStrategy = api.PlanStrategy{
		Continuous:   true,             // User wants continuous charging
		Precondition: 15 * time.Minute, // Standard precondition
	}

	// Execute
	active := lp.plannerActive()

	// Verify: Should CONTINUE charging through 45-minute constructed gap
	// Planner sees three expensive slots and prefers waiting until 05:00 (usually equal costs at later time)
	// But continuous mode overrides: keep charging to avoid interruption
	// returns true (strategy.Continuous && requiredDuration > precondition)
	assert.True(t, active, "Should continue charging in continuous mode through expensive gap (real scenario from logs)")
}

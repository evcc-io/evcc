package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// regression test for https://github.com/evcc-io/evcc/issues/31576
func TestPlannerActiveStopsWhenPlanMovedFarOut(t *testing.T) {
	Voltage = 230

	ctrl := gomock.NewController(t)
	now := time.Now()

	// flat tariff- every 15m slot has the same cost
	var rates api.Rates
	for start := now.Add(-time.Hour); start.Before(now.Add(6 * 24 * time.Hour)); start = start.Add(15 * time.Minute) {
		rates = append(rates, api.Rate{Start: start, End: start.Add(15 * time.Minute), Value: 0.29})
	}

	tariff := api.NewMockTariff(ctrl)
	tariff.EXPECT().Rates().AnyTimes().Return(rates, nil)

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.status = api.StatusC
	lp.planner = planner.New(lp.log, tariff)

	// flat tariff -> planner prefers the latest slot, right before the target
	lp.planTime = now.Add(5 * 24 * time.Hour)
	lp.planEnergy = 5 // kWh, ~27min at 11kW

	// simulate a previously active slot (e.g. from PV surplus) that is about to end
	lp.planActive = true
	lp.planSlotEnd = now.Add(2 * time.Minute)

	assert.False(t, lp.plannerActive(), "must not force continuation once the plan has moved days into the future")
}

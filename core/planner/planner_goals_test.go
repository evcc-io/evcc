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

func TestSubtractRates(t *testing.T) {
	clock := clock.NewMock()
	rr := rates([]float64{10, 20}, clock.Now(), time.Hour)

	// window spanning both slots leaves the outer parts
	res := subtractRates(rr, api.Rates{{Start: clock.Now().Add(30 * time.Minute), End: clock.Now().Add(90 * time.Minute)}})
	require.Len(t, res, 2)
	assert.Equal(t, api.Rate{Start: clock.Now(), End: clock.Now().Add(30 * time.Minute), Value: 10}, res[0])
	assert.Equal(t, api.Rate{Start: clock.Now().Add(90 * time.Minute), End: clock.Now().Add(2 * time.Hour), Value: 20}, res[1])

	// non-overlapping window is a no-op
	assert.Equal(t, rr, subtractRates(rr, api.Rates{{Start: clock.Now().Add(3 * time.Hour), End: clock.Now().Add(4 * time.Hour)}}))

	// window covering everything removes all slots
	assert.Empty(t, subtractRates(rr, api.Rates{{Start: clock.Now(), End: clock.Now().Add(2 * time.Hour)}}))
}

func TestPlanGoals(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{20, 60, 10, 80, 40, 90}, clock.Now(), time.Hour), nil)

	p := &Planner{
		log:    util.NewLogger("foo"),
		clock:  clock,
		tariff: trf,
	}

	// the second goal's cumulative duration is partly covered by the first goal's plan,
	// its shortfall may use cheap slots before the first goal's target time
	plan := p.PlanGoals([]api.PlanGoal{
		{Duration: time.Hour, Time: clock.Now().Add(3 * time.Hour)},
		{Duration: 3 * time.Hour, Time: clock.Now().Add(6 * time.Hour)},
	}, 0, false)

	require.Len(t, plan, 3)
	assert.Equal(t, 3*time.Hour, Duration(plan))
	assert.Equal(t, []float64{20, 10, 40}, []float64{plan[0].Value, plan[1].Value, plan[2].Value})

	// an already covered goal adds no slots
	plan = p.PlanGoals([]api.PlanGoal{
		{Duration: time.Hour, Time: clock.Now().Add(3 * time.Hour)},
		{Duration: time.Hour, Time: clock.Now().Add(6 * time.Hour)},
	}, 0, false)
	assert.Equal(t, time.Hour, Duration(plan))

	// no charging during the first goal's absence window
	plan = p.PlanGoals([]api.PlanGoal{
		{Duration: time.Hour, Time: clock.Now().Add(2 * time.Hour), Absence: 2 * time.Hour},
		{Duration: 2 * time.Hour, Time: clock.Now().Add(6 * time.Hour)},
	}, 0, false)

	require.Len(t, plan, 2)
	assert.Equal(t, []float64{20, 40}, []float64{plan[0].Value, plan[1].Value})
	for _, slot := range plan {
		if slot.Start.Before(clock.Now().Add(4*time.Hour)) && slot.End.After(clock.Now().Add(2*time.Hour)) {
			t.Errorf("slot %v-%v overlaps absence window", slot.Start, slot.End)
		}
	}
}

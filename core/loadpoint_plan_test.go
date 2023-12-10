package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rates(prices []float64, start time.Time, slotDuration time.Duration) api.Rates {
	res := make(api.Rates, 0, len(prices))

	for i, v := range prices {
		slotStart := start.Add(time.Duration(i) * time.Hour)
		ar := api.Rate{
			Start: slotStart,
			End:   slotStart.Add(slotDuration),
			Price: v,
		}
		res = append(res, ar)
	}

	return res
}

func TestGetPlanAfterTargetTime(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{0, 0, 0, 0}, clock.Now(), time.Hour), nil)

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.clock = clock
	lp.planner = planner.New(lp.log, trf)

	{
		// target time +1 hour, no active slot
		ts := clock.Now().Add(time.Hour)
		lp.setPlanEnergy(ts, 2)

		d, r, err := lp.GetPlan(ts, 1e3)
		require.NoError(t, err)
		assert.Equal(t, 2*time.Hour, d)
		assert.Len(t, r, 0)
	}

	{
		// target time now, no active slot
		ts := clock.Now()
		lp.setPlanEnergy(ts, 2)

		d, r, err := lp.GetPlan(ts, 1e3)
		require.NoError(t, err)
		assert.Equal(t, 2*time.Hour, d)
		assert.Len(t, r, 0)
	}

	{
		// target time now, active slot
		ts := clock.Now()
		lp.setPlanEnergy(ts, 2)

		lp.planActive = true

		d, r, err := lp.GetPlan(ts, 1e3)
		require.NoError(t, err)
		assert.Equal(t, 2*time.Hour, d)
		assert.Len(t, r, 0)
	}
}

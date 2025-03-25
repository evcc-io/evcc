package tariff

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeRatesAfter(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	rate := func(start int, val float64) api.Rate {
		return api.Rate{
			Start: clock.Now().Add(time.Duration(start) * time.Hour),
			End:   clock.Now().Add(time.Duration(start+1) * time.Hour),
			Value: val,
		}
	}

	old := api.Rates{rate(1, 1), rate(2, 2)}
	new := api.Rates{rate(2, 2), rate(3, 3)}
	combined := api.Rates{rate(1, 1), rate(2, 2), rate(3, 3)}

	data := util.NewMonitor[api.Rates](time.Hour)
	data.WithClock(clock)

	data.Set(old)
	res, err := data.Get()
	require.NoError(t, err)
	assert.Equal(t, old, res)

	for i, tc := range []struct {
		new, expected api.Rates
		ts            time.Time
	}{
		{new, combined, clock.Now()},
		{nil, combined, clock.Now()},
		{new, combined, clock.Now().Add(time.Hour)},
		{new, combined, now.With(clock.Now().Add(time.Hour + 30*time.Minute)).BeginningOfHour()},
		{new, new, clock.Now().Add(2 * time.Hour)},
	} {
		t.Logf("%d. %+v", i+1, tc)

		mergeRatesAfter(data, tc.new, tc.ts)

		res, err := data.Get()
		require.NoError(t, err)
		assert.Equal(t, tc.expected, res)
	}
}

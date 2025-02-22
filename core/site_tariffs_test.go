package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/assert"
)

func TestAccumulatedEnergy(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	rate := func(start int, val float64) api.Rate {
		return api.Rate{
			Start: clock.Now().Add(time.Duration(start) * time.Hour),
			End:   clock.Now().Add(time.Duration(start+1) * time.Hour),
			Price: val,
		}
	}

	rr := api.Rates{rate(0, 0), rate(1, 1), rate(2, 2), rate(3, 3), rate(4, 4)}

	for i, tc := range []struct {
		from, to float64
		expected float64
	}{
		{0, 0, 0},
		{0, 0.5, 0.25},
		{0, 1, 0.5},
		{1, 2, 1.5},
		{0, 2, 2},
	} {
		t.Logf("%d. %+v", i+1, tc)

		from := clock.Now().Add(time.Duration(float64(time.Hour) * tc.from))
		to := clock.Now().Add(time.Duration(float64(time.Hour) * tc.to))

		res := accumulatedEnergy(rr, from, to)
		assert.Equal(t, tc.expected, res, "test case %d", i)
	}
}

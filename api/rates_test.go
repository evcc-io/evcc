package api

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
)

func TestRates(t *testing.T) {
	clock := clock.NewMock()
	rate := func(start int, val float64) Rate {
		return Rate{
			Start: clock.Now().Add(time.Duration(start) * time.Hour),
			End:   clock.Now().Add(time.Duration(start+1) * time.Hour),
			Price: val,
		}
	}

	rr := Rates{rate(1, 1), rate(2, 2), rate(3, 3), rate(4, 4)}

	_, err := rr.At(clock.Now())
	assert.Error(t, err)

	for i := 1; i <= 4; i++ {
		r, err := rr.At(clock.Now().Add(time.Duration(i) * time.Hour))
		assert.NoError(t, err)
		assert.Equal(t, float64(i), r.Price)

		r, err = rr.At(clock.Now().Add(time.Duration(i)*time.Hour + 30*time.Minute))
		assert.NoError(t, err)
		assert.Equal(t, float64(i), r.Price)
	}

	_, err = rr.At(clock.Now().Add(5 * time.Hour))
	assert.Error(t, err)
}

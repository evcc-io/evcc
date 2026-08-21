package tariff

import (
	"errors"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tariff struct {
	rates api.Rates
	err   error
}

func (t *tariff) Rates() (api.Rates, error) {
	return t.rates, t.err
}

func (t *tariff) Type() api.TariffType {
	return api.TariffTypeSolar
}

func TestCombined(t *testing.T) {
	clock := clock.NewMock()
	rate := func(start int, val float64) api.Rate {
		return api.Rate{
			Start: clock.Now().Add(time.Duration(start) * time.Hour),
			End:   clock.Now().Add(time.Duration(start+1) * time.Hour),
			Value: val,
		}
	}

	a := &tariff{rates: api.Rates{rate(1, 1), rate(2, 2)}}
	b := &tariff{rates: api.Rates{rate(2, 2), rate(3, 3)}}
	c := &combined{[]api.Tariff{a, b}}

	rr, err := c.Rates()
	require.NoError(t, err)
	assert.Equal(t, api.Rates{rate(1, 1), rate(2, 4), rate(3, 3)}, rr)
}

func TestCombinedUnsorted(t *testing.T) {
	clock := clock.NewMock()
	rate := func(start int, val float64) api.Rate {
		return api.Rate{
			Start: clock.Now().Add(time.Duration(start) * time.Hour),
			End:   clock.Now().Add(time.Duration(start+1) * time.Hour),
			Value: val,
		}
	}

	// b covers a disjoint range not adjacent to a's rates after concatenation
	a := &tariff{rates: api.Rates{rate(1, 1), rate(2, 2)}}
	b := &tariff{rates: api.Rates{rate(3, 3)}}
	c := &tariff{rates: api.Rates{rate(1, 10), rate(2, 20)}}
	comb := &combined{[]api.Tariff{a, b, c}}

	rr, err := comb.Rates()
	require.NoError(t, err)
	assert.Equal(t, api.Rates{rate(1, 11), rate(2, 22), rate(3, 3)}, rr)
}

func TestCombinedPartialError(t *testing.T) {
	clock := clock.NewMock()
	rate := func(start int, val float64) api.Rate {
		return api.Rate{
			Start: clock.Now().Add(time.Duration(start) * time.Hour),
			End:   clock.Now().Add(time.Duration(start+1) * time.Hour),
			Value: val,
		}
	}

	// a source that is temporarily unavailable must not discard the other sources' rates
	a := &tariff{rates: api.Rates{rate(1, 1), rate(2, 2)}}
	b := &tariff{err: api.ErrOutdated}
	c := &combined{[]api.Tariff{a, b}}

	rr, err := c.Rates()
	require.NoError(t, err)
	assert.Equal(t, api.Rates{rate(1, 1), rate(2, 2)}, rr)
}

func TestCombinedAllErrors(t *testing.T) {
	a := &tariff{err: api.ErrOutdated}
	b := &tariff{err: api.ErrOutdated}
	c := &combined{[]api.Tariff{a, b}}

	_, err := c.Rates()
	require.Error(t, err)
	require.True(t, errors.Is(err, api.ErrOutdated))
}

func BenchmarkCombined(bench *testing.B) {
	clock := clock.NewMock()
	rate := func(start int, val float64) api.Rate {
		return api.Rate{
			Start: clock.Now().Add(time.Duration(start) * time.Hour),
			End:   clock.Now().Add(time.Duration(start+1) * time.Hour),
			Value: val,
		}
	}

	a := &tariff{rates: api.Rates{rate(1, 1), rate(2, 2)}}
	b := &tariff{rates: api.Rates{rate(2, 2), rate(3, 3)}}
	c := &combined{[]api.Tariff{a, b}}

	for bench.Loop() {
		c.Rates()
	}
}

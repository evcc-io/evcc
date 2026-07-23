package tariff

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tariff struct {
	rates api.Rates
}

func (t *tariff) Rates() (api.Rates, error) {
	return t.rates, nil
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

	a := &tariff{api.Rates{rate(1, 1), rate(2, 2)}}
	b := &tariff{api.Rates{rate(2, 2), rate(3, 3)}}
	c := &combined{[]api.Tariff{a, b}}

	rr, err := c.Rates()
	require.NoError(t, err)
	assert.Equal(t, api.Rates{rate(1, 1), rate(2, 4), rate(3, 3)}, rr)
}

// TestCombinedNonAdjacent verifies that rates are correctly summed
// even when tariffs cover different but overlapping time ranges,
// causing identical timestamps to be non-adjacent after concatenation.
func TestCombinedNonAdjacent(t *testing.T) {
	clock := clock.NewMock()
	rate := func(start int, val float64) api.Rate {
		return api.Rate{
			Start: clock.Now().Add(time.Duration(start) * time.Hour),
			End:   clock.Now().Add(time.Duration(start+1) * time.Hour),
			Value: val,
		}
	}

	// tariff a covers hours 1-3, tariff b covers hours 2-4
	// after append: [h1a, h2a, h3a, h2b, h3b, h4b]
	// without sort: PartitionBy sees h2a and h2b as separate groups
	a := &tariff{api.Rates{rate(1, 100), rate(2, 200), rate(3, 300)}}
	b := &tariff{api.Rates{rate(2, 20), rate(3, 30), rate(4, 40)}}
	c := &combined{[]api.Tariff{a, b}}

	rr, err := c.Rates()
	require.NoError(t, err)

	expected := api.Rates{
		rate(1, 100),
		rate(2, 220),
		rate(3, 330),
		rate(4, 40),
	}
	assert.Equal(t, expected, rr)
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

	a := &tariff{api.Rates{rate(1, 1), rate(2, 2)}}
	b := &tariff{api.Rates{rate(2, 2), rate(3, 3)}}
	c := &combined{[]api.Tariff{a, b}}

	for bench.Loop() {
		c.Rates()
	}
}

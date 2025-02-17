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
			Price: val,
		}
	}

	a := &tariff{api.Rates{rate(1, 1), rate(2, 2)}}
	b := &tariff{api.Rates{rate(2, 2), rate(3, 3)}}
	c := &combined{[]api.Tariff{a, b}}

	rr, err := c.Rates()
	require.NoError(t, err)
	assert.Equal(t, api.Rates{rate(1, 1), rate(2, 4), rate(3, 3)}, rr)
}

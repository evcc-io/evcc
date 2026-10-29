package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSolarRates(t *testing.T) {
	suite.Run(t, new(solarTestSuite))
}

type solarTestSuite struct {
	suite.Suite
	clock *clock.Mock
	rr    api.Rates
}

func (t *solarTestSuite) rate(start int, val float64) api.Rate {
	return api.Rate{
		Start: t.clock.Now().Add(time.Duration(start) * time.Hour),
		End:   t.clock.Now().Add(time.Duration(start+1) * time.Hour),
		Value: val,
	}
}

func (t *solarTestSuite) SetupSuite() {
	t.clock = clock.NewMock()
	t.clock.Set(now.BeginningOfDay())
	t.rr = api.Rates{t.rate(0, 0), t.rate(1, 1), t.rate(2, 2), t.rate(3, 3), t.rate(4, 4)}
}

func (t *solarTestSuite) TestIndex() {
	for i, tc := range []struct {
		ts  float64
		idx int
		ok  bool
	}{
		{-1, 0, false},
		{0, 0, true},
		{0.5, 1, false},
		{1, 1, true},
		{99, len(t.rr), false},
	} {
		ts := t.clock.Now().Add(time.Duration(float64(time.Hour) * tc.ts))
		res, ok := search(t.rr, ts)
		t.Equal(tc.idx, res, "%d. idx %+v", i+1, tc)
		t.Equal(tc.ok, ok, "%d. ok %+v", i+1, tc)
	}
}

func (t *solarTestSuite) TestEnergy() {
	for i, tc := range []struct {
		from, to float64
		expected float64
	}{
		{-1, 0, 0},
		{-2, -1, 0},   // whole interval before first entry
		{-1, -0.5, 0}, // whole interval before first entry
		{-1, 1, 0.5},
		{-1, 90, 8},
		{0, 0, 0},
		{0, 0.5, 0.125},
		{0, 1, 0.5},
		{0, 1.5, 1.125},
		{0, 2, 2},
		{1, 2, 1.5},
		{0.25, 0.75, 0.25},
		{0.5, 1, 0.375},
		{0.5, 3.5, 6},
		{80, 90, 0},
	} {
		from := t.clock.Now().Add(time.Duration(float64(time.Hour) * tc.from))
		to := t.clock.Now().Add(time.Duration(float64(time.Hour) * tc.to))

		res := solarEnergy(t.rr, from, to)
		t.Equal(tc.expected, res, "%d. %+v", i+1, tc)
	}
}

func (t *solarTestSuite) TestShort() {
	t.clock.Set(now.BeginningOfDay())
	rr := api.Rates{t.rate(0, 0), t.rate(1, 1)}

	for i, tc := range []struct {
		from, to, energy, value float64
	}{
		{-1, 0, 0, 0},
		// {-1, 0.5, 0.125, 0.5},
		// {-1, 2, 0.5, 0},
		{0, 0, 0, 0},
		{0, 0.5, 0.125, 0.5},
		{0, 1, 0.5, 1},
		{0, 1.5, 0.5, 0},
		{1.5, 2, 0, 0},
	} {
		from := t.clock.Now().Add(time.Duration(float64(time.Hour) * tc.from))
		to := t.clock.Now().Add(time.Duration(float64(time.Hour) * tc.to))

		t.Equal(tc.energy, solarEnergy(rr, from, to), "%d. energy %+v", i+1, tc)
	}
}

func TestSolarEnergyNoRates(t *testing.T) {
	// empty rate series must not panic and yields no energy
	now := time.Now()
	assert.Equal(t, 0.0, solarEnergy(api.Rates{}, now, now.Add(time.Hour)))
	assert.Equal(t, 0.0, solarEnergy(nil, now, now.Add(time.Hour)))
}

type solarTariff struct {
	rates api.Rates
}

func (t *solarTariff) Rates() (api.Rates, error) { return t.rates, nil }
func (t *solarTariff) Type() api.TariffType      { return api.TariffTypeSolar }

// TestSolarSlotSplitPreservesEnergy asserts that splitting hourly solar rates into
// 15min slots leaves the integrated energy untouched- the wrapper resamples the
// curve, it does not change it
func TestSolarSlotSplitPreservesEnergy(t *testing.T) {
	start := time.Now().Truncate(tariff.SlotDuration)

	var rr api.Rates
	for i, v := range []float64{0, 1000, 2500, 3000, 1500, 0} {
		s := start.Add(time.Duration(i) * time.Hour)
		rr = append(rr, api.Rate{Start: s, End: s.Add(time.Hour), Value: v})
	}

	w := &tariff.SlotWrapper{Tariff: &solarTariff{rates: rr}}
	res, err := w.Rates()
	require.NoError(t, err)
	require.Len(t, res, 6*4)

	to := rr[len(rr)-1].Start
	assert.InDelta(t, solarEnergy(rr, start, to), solarEnergy(res, start, to), 1e-9)

	// every sub-interval integrates identically, not just the total
	for i := range res {
		assert.InDelta(t,
			solarEnergy(rr, start, res[i].Start),
			solarEnergy(res, start, res[i].Start),
			1e-9, "slot %d", i)
	}
}

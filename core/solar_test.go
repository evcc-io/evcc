package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/assert"
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

func (t *solarTestSuite) TestEnergy() {
	for i, tc := range []struct {
		from, to float64
		expected float64
	}{
		{-1, 0, 0},
		{-2, -1, 0},   // whole interval before first entry
		{-1, -0.5, 0}, // whole interval before first entry
		{-1, 1, 0},
		{-1, 90, 10}, // all slots
		{0, 0, 0},
		{0, 2, 1},
		{1, 2, 1},
		{1.5, 2, 0.5},     // half of second slot
		{1.25, 1.75, 0.5}, // middle of second slot
		{1.5, 2.5, 1.5},   // half of second and third slot
		{0.5, 3.5, 4.5},
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
		from, to, energy float64
	}{
		{-1, 0, 0},
		{0, 0, 0},
		{0, 1, 0},
		{0, 1.5, 0.5},
		{1.5, 2, 0.5},
		{2, 3, 0},
	} {
		from := t.clock.Now().Add(time.Duration(float64(time.Hour) * tc.from))
		to := t.clock.Now().Add(time.Duration(float64(time.Hour) * tc.to))

		t.Equal(tc.energy, solarEnergy(rr, from, to), "%d. energy %+v", i+1, tc)
	}
}

func (t *solarTestSuite) TestTimeseries() {
	// energy per slot is converted back to average power
	rr := api.Rates{t.rate(0, 500), {
		Start: t.clock.Now().Add(time.Hour),
		End:   t.clock.Now().Add(time.Hour + 15*time.Minute),
		Value: 500,
	}}

	ts := solarTimeseries(rr)
	t.Require().Len(ts, 2)
	t.Equal(500.0, ts[0].Value)
	t.Equal(2000.0, ts[1].Value)
}

func TestSolarEnergyNoRates(t *testing.T) {
	// empty rate series must not panic and yields no energy
	now := time.Now()
	assert.Equal(t, 0.0, solarEnergy(api.Rates{}, now, now.Add(time.Hour)))
	assert.Equal(t, 0.0, solarEnergy(nil, now, now.Add(time.Hour)))
}

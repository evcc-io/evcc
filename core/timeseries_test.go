package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/suite"
)

func TestRates(t *testing.T) {
	suite.Run(t, new(ratesTestSuite))
}

type ratesTestSuite struct {
	suite.Suite
	clock *clock.Mock
	rr    api.Rates
}

func (t *ratesTestSuite) rate(start int, val float64) api.Rate {
	return api.Rate{
		Start: t.clock.Now().Add(time.Duration(start) * time.Hour),
		End:   t.clock.Now().Add(time.Duration(start+1) * time.Hour),
		Value: val,
	}
}

func (t *ratesTestSuite) SetupSuite() {
	t.clock = clock.NewMock()
	t.clock.Set(now.BeginningOfDay())
	t.rr = api.Rates{t.rate(0, 0), t.rate(1, 1), t.rate(2, 2), t.rate(3, 3), t.rate(4, 4)}
}

func (t *ratesTestSuite) TestIndex() {
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

func (t *ratesTestSuite) TestValue() {
	for i, tc := range []struct {
		ts, val float64
	}{
		{-1, 0},
		{0, 0},
		{0.5, 0.5},
		{1, 1},
		{4, 4},
		{99, 0},
	} {
		ts := t.clock.Now().Add(time.Duration(float64(time.Hour) * tc.ts))
		res := value(t.rr, ts)
		t.Equal(tc.val, res, "%d. %+v", i+1, tc)
	}
}

func (t *ratesTestSuite) TestEnergy() {
	for i, tc := range []struct {
		from, to float64
		expected float64
	}{
		{-1, 0, 0},
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

		res := energy(t.rr, from, to)
		t.Equal(tc.expected, res, "%d. %+v", i+1, tc)
	}
}

func (t *ratesTestSuite) TestShort() {
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

		t.Equal(tc.energy, energy(rr, from, to), "%d. energy %+v", i+1, tc)
		t.Equal(tc.value, value(rr, to), "%d. value %+v", i+1, tc)
	}
}

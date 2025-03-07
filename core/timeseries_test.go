package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/suite"
)

func TestTimeseries(t *testing.T) {
	suite.Run(t, new(timeseriesTestSuite))
}

type timeseriesTestSuite struct {
	suite.Suite
	clock *clock.Mock
	rr    timeseries
}

func (t *timeseriesTestSuite) SetupSuite() {
	t.clock = clock.NewMock()
	t.clock.Set(now.BeginningOfDay())

	rate := func(start int, val float64) tsval {
		return tsval{
			Timestamp: t.clock.Now().Add(time.Duration(start) * time.Hour),
			Value:     val,
		}
	}

	t.rr = timeseries{rate(0, 0), rate(1, 1), rate(2, 2), rate(3, 3), rate(4, 4)}
}

func (t *timeseriesTestSuite) TestIndex() {
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
		res, ok := t.rr.index(ts)
		t.Equal(tc.idx, res, "%d. %+v idx", i+1, tc)
		t.Equal(tc.ok, ok, "%d. %+v ok", i+1, tc)
	}
}

func (t *timeseriesTestSuite) TestAccumulatedEnergy() {
	for i, tc := range []struct {
		from, to float64
		expected float64
	}{
		{0, 0, 0},
		{0, 0.5, 0.125},
		{0, 1, 0.5},
		{0, 1.5, 1.125},
		{0, 2, 2},
		{1, 2, 1.5},
		{0.25, 0.75, 0.25},
		{0.5, 1, 0.375},
		{0.5, 3.5, 6},
	} {
		t.T().Logf("%d. %+v", i+1, tc)

		from := t.clock.Now().Add(time.Duration(float64(time.Hour) * tc.from))
		to := t.clock.Now().Add(time.Duration(float64(time.Hour) * tc.to))

		res := t.rr.energy(from, to)
		t.Equal(tc.expected, res, "test case %d", i+1)
	}
}

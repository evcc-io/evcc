package provider

import (
	"errors"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestCachedGetter(t *testing.T) {
	var idx int
	cases := []struct {
		f float64
		e error
	}{
		{f: 1, e: nil},
		{f: 2, e: nil},
		{f: 3, e: errors.New("3")},
	}

	g := func() (float64, error) {
		f := cases[idx].f
		e := cases[idx].e
		idx++
		return f, e
	}

	duration := time.Second
	c := ResettableCached(g, duration)
	clock := clock.NewMock()
	c.clock = clock

	expect := func(s struct {
		f float64
		e error
	},
	) {
		f, e := c.Get()
		if f != s.f || e != s.e {
			t.Errorf("unexpected cache value: %f, %v\n", f, e)
		}
	}

	expect(cases[0])
	expect(cases[0])

	clock.Add(2 * duration)
	expect(cases[1])

	clock.Add(2 * duration)
	expect(cases[2])
}

func TestCacheReset(t *testing.T) {
	var i int64
	g := func() (int64, error) {
		i++
		return i, nil
	}

	c := ResettableCached(g, 10*time.Minute)
	clock := clock.NewMock()
	c.clock = clock

	test := func(exp int64) {
		v, _ := c.Get()
		if exp != v {
			t.Errorf("expected %d, got %d", exp, v)
		}
	}

	test(1)
	test(1)
	c.Reset()
	test(2)
	test(2)
	clock.Add(10*time.Minute + 1)
	test(3)
}

func TestRetryWithBackoff(t *testing.T) {
	tests := []struct {
		deltaTime      time.Duration
		returnError    bool
		functionCalled bool
	}{
		{0 * time.Second, true, true},
		{4 * time.Second, true, false},
		{6 * time.Second, true, true},
		{9 * time.Second, true, false},
		{11 * time.Second, true, true},
		{14 * time.Second, true, false},
		{16 * time.Second, false, true},
		{16 * time.Minute, true, true},
		{4 * time.Second, true, false},
		{6 * time.Second, true, true},
	}

	var returnError, functionCalled bool

	g := func() (int64, error) {
		functionCalled = true
		if returnError {
			return 1, api.ErrTimeout
		}
		return 1, nil
	}

	c := ResettableCached(g, 15*time.Minute)
	clock := clock.NewMock()
	c.clock = clock

	for _, tt := range tests {
		functionCalled = false
		returnError = tt.returnError

		clock.Add(tt.deltaTime)

		_, err := c.Get()

		if returnError {
			assert.Error(t, err)
		}

		assert.Equal(t, tt.functionCalled, functionCalled)
	}
}

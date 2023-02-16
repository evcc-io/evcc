package provider

import (
	"errors"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
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
	}) {
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

// nolint:errcheck
func TestRetryWithBackoff(t *testing.T) {

	cacheTime := time.Minute * 10

	clock := clock.NewMock()
	clock.Set(time.Now())

	returnError := true

	functionCalled := false

	g := func() (float64, error) {
		functionCalled = true
		if returnError {
			return float64(1), errors.New("timeout")
		} else {
			return float64(1), nil
		}
	}

	c := ResettableCached(g, cacheTime)
	c.clock = clock

	// Put cache in error state
	c.Get()

	// Half a minute later -> cache time is 10 minutes, so no regular get. No retry because of backoff wait time after first error is 1m.
	functionCalled = false
	clock.Add(30 * time.Second)
	c.Get()

	if functionCalled {
		t.Errorf("unexpected function call")
	}

	// Total expired time is now 1m30s. No regular get but retry.
	clock.Add(60 * time.Second)
	c.Get()

	if !functionCalled {
		t.Errorf("expected function call not executed")
	}

	// Still in error state after first retry. Backoff wait time is now 2m.
	// 1m30s later: no retry.
	functionCalled = false
	clock.Add(90 * time.Second)
	c.Get()

	if functionCalled {
		t.Errorf("unexpected function call")
	}

	// Still in error state after first retry. Backoff wait time is still 2m.
	// 3m later: retry.
	clock.Add(90 * time.Second)
	c.Get()

	if !functionCalled {
		t.Errorf("expected function call not executed")
	}

	// Cache getter returned error again. Backoff wait time now at 4m.
	// No retry because of backoff wait time.
	functionCalled = false
	clock.Add(3 * time.Minute)
	c.Get()

	if functionCalled {
		t.Errorf("unexpected function call")
	}

	// 5m later: retry. Cache getter will not return an error this time -> backoff will be reset.
	returnError = false
	clock.Add(5 * time.Minute)
	c.Get()

	if !functionCalled {
		t.Errorf("expected function call not executed")
	}

	// Put cache in error state again with regular get after 11 minutes (10 minutes cache time)
	returnError = true
	clock.Add(11 * time.Minute)
	c.Get()

	// Half a minute later -> cache time is 10 minutes, so no regular get. No retry because of backoff wait time after first error is 1m.
	functionCalled = false
	clock.Add(30 * time.Second)
	c.Get()

	if functionCalled {
		t.Errorf("unexpected function call")
	}

	// Total expired time is now 1m30s. No regular get but retry.
	clock.Add(60 * time.Second)
	c.Get()

	if !functionCalled {
		t.Errorf("expected function call not executed")
	}
}

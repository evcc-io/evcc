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
	c := NewCached(g, duration)
	clck := clock.NewMock()
	c.clock = clck
	getter := c.FloatGetter()

	expect := func(s struct {
		f float64
		e error
	}) {
		f, e := getter()
		if f != s.f || e != s.e {
			t.Errorf("unexpected cache value: %f, %v\n", f, e)
		}
	}

	expect(cases[0])
	expect(cases[0])

	clck.Add(2 * duration)
	expect(cases[1])

	clck.Add(2 * duration)
	expect(cases[2])
}

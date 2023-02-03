package server

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
	inf2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/stretchr/testify/assert"
)

type influxWriter struct {
	t   *testing.T
	p   []*write.Point
	idx int
}

func (w *influxWriter) WritePoint(p *write.Point) {
	if w.idx >= len(w.p) {
		w.t.Fatal("too many points")
	}

	assert.Equal(w.t, w.p[w.idx], p)
	w.idx++
}

func (w *influxWriter) finish() {
	assert.Equal(w.t, len(w.p), w.idx, "not enough points")
}

func TestInfluxTypes(t *testing.T) {
	m := &Influx{
		log:   util.NewLogger("foo"),
		clock: clock.NewMock(),
	}

	{
		// string value
		w := &influxWriter{
			t: t, p: []*write.Point{inf2.NewPoint("foo", nil, map[string]any{"value": 1}, m.clock.Now())},
		}
		m.writeComplexPoint(w, util.Param{Key: "foo", Val: 1}, nil)
		w.finish()
	}

	{
		// nil value - https://github.com/evcc-io/evcc/issues/5950
		w := &influxWriter{
			t: t, p: []*write.Point{inf2.NewPoint("phasesConfigured", nil, map[string]any{"value": nil}, m.clock.Now())},
		}
		m.writeComplexPoint(w, util.Param{Key: "phasesConfigured", Val: nil}, nil)
		w.finish()
	}

	{
		// phases array
		w := &influxWriter{
			t: t, p: []*write.Point{inf2.NewPoint("foo", nil, map[string]any{
				"l1": 1.0,
				"l2": 2.0,
				"l3": 3.0,
			}, m.clock.Now())},
		}
		m.writeComplexPoint(w, util.Param{Key: "foo", Val: [3]float64{1, 2, 3}}, nil)
		w.finish()
	}

	{
		// phases slice
		w := &influxWriter{
			t: t, p: []*write.Point{inf2.NewPoint("foo", nil, map[string]any{
				"l1": 1.0,
				"l2": 2.0,
				"l3": 3.0,
			}, m.clock.Now())},
		}
		m.writeComplexPoint(w, util.Param{Key: "foo", Val: []float64{1, 2, 3}}, nil)
		w.finish()
	}

	{
		// arbitrary slice
		w := &influxWriter{
			t: t, p: nil,
		}
		m.writeComplexPoint(w, util.Param{Key: "foo", Val: []float64{1, 2, 3, 4}}, nil)
		w.finish()
	}
}

package server

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
	inf2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
)

func TestInfluxTypes(t *testing.T) {
	suite.Run(t, new(influxSuite))
}

type influxSuite struct {
	suite.Suite
	*Influx
	p []*write.Point
}

func (suite *influxSuite) SetupSuite() {
	suite.Influx = &Influx{
		log:   util.NewLogger("foo"),
		clock: clock.NewMock(),
	}
}

func (suite *influxSuite) SetupTest() {
	suite.p = nil
}

func (suite *influxSuite) WritePoint(p *write.Point) {
	suite.p = append(suite.p, p)
}

func (suite *influxSuite) WriteParam(p util.Param) {
	tags := make(map[string]string)
	suite.Influx.writeComplexPoint(suite, p.Key, p.Val, tags)
}

func (w *influxSuite) TestString() {
	w.WriteParam(util.Param{Key: "foo", Val: 1})
	w.Equal([]*write.Point{inf2.NewPoint("foo", nil, map[string]any{"value": 1}, w.clock.Now())}, w.p)
}

// bool is not published
// func (w *influxSuite) TestBool() {
// 	w.WriteParam(util.Param{Key: "foo", Val: false})
// 	w.Equal([]*write.Point{inf2.NewPoint("foo", nil, map[string]any{"value": "false"}, w.clock.Now())}, w.p)
// }

func (w *influxSuite) TestNil() {
	// nil value - https://github.com/evcc-io/evcc/issues/5950
	w.WriteParam(util.Param{Key: "foo", Val: nil})
	w.Equal([]*write.Point{inf2.NewPoint("foo", nil, map[string]any{"value": nil}, w.clock.Now())}, w.p)
}

func (w *influxSuite) TestPointer() {
	w.WriteParam(util.Param{Key: "foo", Val: lo.ToPtr(1)})
	w.Equal([]*write.Point{inf2.NewPoint("foo", nil, map[string]any{"value": 1}, w.clock.Now())}, w.p)
}

func (w *influxSuite) TestArray() {
	// nil value - https://github.com/evcc-io/evcc/issues/5950
	w.WriteParam(util.Param{Key: "foo", Val: [3]float64{1, 2, 3}})
	w.Equal([]*write.Point{inf2.NewPoint("foo", nil, map[string]any{
		"l1": 1.0,
		"l2": 2.0,
		"l3": 3.0,
	}, w.clock.Now())}, w.p)
}

func (w *influxSuite) TestPhasesSlice() {
	w.WriteParam(util.Param{Key: "foo", Val: []float64{1, 2, 3}})
	w.Equal([]*write.Point{inf2.NewPoint("foo", nil, map[string]any{
		"l1": 1.0,
		"l2": 2.0,
		"l3": 3.0,
	}, w.clock.Now())}, w.p)
}

func (w *influxSuite) TestSlice() {
	w.WriteParam(util.Param{Key: "foo", Val: []float64{1, 2, 3, 4}})
	w.Len(w.p, 0)
}

func (w *influxSuite) TestMeasurement() {
	w.WriteParam(util.Param{Key: "battery", Val: measurement{Power: 1, Soc: lo.ToPtr(10.0)}})
	w.Equal([]*write.Point{
		inf2.NewPoint("batteryPower", nil, map[string]any{"value": 1.0}, w.clock.Now()),
		inf2.NewPoint("batterySoc", nil, map[string]any{"value": 10.0}, w.clock.Now()),
	}, w.p)
}

func (w *influxSuite) TestSliceOfStruct() {
	w.WriteParam(util.Param{Key: "grid", Val: []measurement{
		{Power: 1, Soc: lo.ToPtr(10.0)},
		{Power: 2, Soc: lo.ToPtr(20.0)},
	}})
	w.Equal([]*write.Point{
		inf2.NewPoint("gridPower", map[string]string{"id": "1"}, map[string]any{"value": 1.0}, w.clock.Now()),
		inf2.NewPoint("gridSoc", map[string]string{"id": "1"}, map[string]any{"value": 10.0}, w.clock.Now()),
		inf2.NewPoint("gridPower", map[string]string{"id": "2"}, map[string]any{"value": 2.0}, w.clock.Now()),
		inf2.NewPoint("gridSoc", map[string]string{"id": "2"}, map[string]any{"value": 20.0}, w.clock.Now()),
	}, w.p)
}

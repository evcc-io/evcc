package core

import (
	"reflect"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

type charger struct {
	caps map[reflect.Type]any
}

var _ api.Capable = (*charger)(nil)

func (c *charger) Capability(typ reflect.Type) (any, bool) {
	cap, ok := c.caps[typ]
	if !ok && reflect.TypeOf(c).Implements(typ) {
		return c, true
	}
	return cap, ok
}

var _ api.Meter = (*charger)(nil)

func (c *charger) CurrentPower() (float64, error) {
	return 0, nil
}

var _ api.BatteryCapacity = (*charger)(nil)

func (c *charger) Capacity() float64 {
	return 0
}

var _ api.MeterImport = (*charger)(nil)

func (c *charger) ImportEnergy() (float64, error) {
	return 0, nil
}

var _ api.Battery = (*batteryImpl)(nil)

type batteryImpl struct {
	soc func() (float64, error)
}

func (impl *batteryImpl) Soc() (float64, error) {
	return impl.soc()
}

func TestCapsWrapping(t *testing.T) {
	// type is just a shortcut for something simple that is not a meter
	var c api.BatteryCapacity

	c = &charger{
		caps: make(map[reflect.Type]any),
	}

	c.(*charger).caps[reflect.TypeFor[api.Battery]()] = &batteryImpl{
		soc: func() (float64, error) {
			return 0, nil
		},
	}

	{
		_, ok := c.(api.Meter)
		assert.True(t, ok)
	}
	{
		_, ok := c.(api.MeterImport)
		assert.True(t, ok)
	}
	{
		_, ok := c.(api.Battery)
		assert.False(t, ok)
		assert.True(t, api.HasCap[api.Battery](c))
	}

	var m api.Meter

	if mt, ok := api.Cap[api.Meter](c); ok {
		if c, ok := c.(api.Capable); ok {
			m = &capableMeter{Meter: mt, Capable: c}
		} else {
			m = mt
		}
	}

	{
		_, ok := m.(api.MeterImport)
		assert.False(t, ok, "unexpected promoted energy")
		assert.True(t, api.HasCap[api.MeterImport](m), "missing promoted energy cap")

		var mm any = m.(*capableMeter).Meter
		_, ok = mm.(api.MeterImport)
		assert.True(t, ok, "missing embedded energy")
		assert.True(t, api.HasCap[api.MeterImport](mm), "missing embedded energy cap")
	}
	{
		_, ok := m.(api.Battery)
		assert.False(t, ok)
		assert.True(t, api.HasCap[api.Battery](m), "missing battery cap")
	}
}

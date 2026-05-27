package core

import (
	"reflect"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
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

var _ api.MeterEnergy = (*charger)(nil)

func (c *charger) TotalEnergy() (float64, error) {
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
		_, ok := c.(api.MeterEnergy)
		assert.True(t, ok)
	}
	{
		_, ok := c.(api.Battery)
		assert.False(t, ok)
		assert.True(t, api.HasCap[api.Battery](c))
	}

	var m api.Meter

	if mt, ok := api.Cap[api.Meter](c); ok {
		m = &capableMeter{Meter: mt, source: c}
	}

	{
		_, ok := m.(api.MeterEnergy)
		assert.False(t, ok, "unexpected promoted energy")
		assert.True(t, api.HasCap[api.MeterEnergy](m), "missing promoted energy cap")

		var mm any = m.(*capableMeter).Meter
		_, ok = mm.(api.MeterEnergy)
		assert.True(t, ok, "missing embedded energy")
		assert.True(t, api.HasCap[api.MeterEnergy](mm), "missing embedded energy cap")
	}
	{
		_, ok := m.(api.Battery)
		assert.False(t, ok)
		assert.True(t, api.HasCap[api.Battery](m), "missing battery cap")
	}
}

// staticPhaseCharger emulates a charger like DaheimLaden: it embeds an
// implement.Caps registry (so it satisfies api.Capable) but exposes
// api.Meter and api.PhaseCurrents as static struct methods rather than
// registering them in the registry.
type staticPhaseCharger struct {
	implement.Caps
}

var (
	_ api.Capable       = (*staticPhaseCharger)(nil)
	_ api.Meter         = (*staticPhaseCharger)(nil)
	_ api.PhaseCurrents = (*staticPhaseCharger)(nil)
)

func (*staticPhaseCharger) CurrentPower() (float64, error)               { return 0, nil }
func (*staticPhaseCharger) Currents() (float64, float64, float64, error) { return 1, 2, 3, nil }

// TestCapableMeterStaticInterface guards against the regression in
// https://github.com/evcc-io/evcc/issues/29877: a charger that embeds
// implement.Caps but implements PhaseCurrents as a static method must
// still expose that capability through the capableMeter wrapper.
func TestCapableMeterStaticInterface(t *testing.T) {
	c := &staticPhaseCharger{Caps: implement.New()}

	var m api.Meter
	if mt, ok := api.Cap[api.Meter](c); ok {
		m = &capableMeter{Meter: mt, source: c}
	}

	assert.True(t, api.HasCap[api.PhaseCurrents](m),
		"PhaseCurrents must remain discoverable on capableMeter when implemented statically")

	pc, ok := api.Cap[api.PhaseCurrents](m)
	assert.True(t, ok)
	i1, i2, i3, err := pc.Currents()
	assert.NoError(t, err)
	assert.Equal(t, []float64{1, 2, 3}, []float64{i1, i2, i3})
}

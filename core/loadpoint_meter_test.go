package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registryCharger exposes all capabilities through the registry
type registryCharger struct {
	implement.Caps
}

// staticPhaseCharger emulates a charger like DaheimLaden: it embeds an
// implement.Caps registry but exposes api.Meter and api.PhaseCurrents
// as static struct methods rather than registering them.
type staticPhaseCharger struct {
	implement.Caps
}

func (*staticPhaseCharger) CurrentPower() (float64, error)               { return 0, nil }
func (*staticPhaseCharger) Currents() (float64, float64, float64, error) { return 1, 2, 3, nil }

// https://github.com/evcc-io/evcc/issues/28915
func TestChargeMeterRegistryCaps(t *testing.T) {
	c := &registryCharger{Caps: implement.New()}
	implement.Has(c.Caps, implement.Meter(func() (float64, error) { return 1, nil }))
	implement.Has(c.Caps, implement.MeterEnergy(func() (float64, error) { return 2, nil }))

	m := newChargeMeter(c)
	assert.Nil(t, m.fake)

	_, ok := any(m).(api.MeterEnergy)
	assert.False(t, ok, "unexpected static energy")

	me, ok := api.Cap[api.MeterEnergy](m)
	require.True(t, ok, "missing registry energy cap")
	f, err := me.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 2.0, f)

	assert.False(t, api.HasCap[api.PhaseCurrents](m))
}

// https://github.com/evcc-io/evcc/issues/29877
func TestChargeMeterStaticCaps(t *testing.T) {
	m := newChargeMeter(&staticPhaseCharger{Caps: implement.New()})
	assert.Nil(t, m.fake)

	pc, ok := api.Cap[api.PhaseCurrents](m)
	require.True(t, ok, "missing static phase currents cap")
	i1, i2, i3, err := pc.Currents()
	require.NoError(t, err)
	assert.Equal(t, []float64{1, 2, 3}, []float64{i1, i2, i3})

	assert.False(t, api.HasCap[api.MeterEnergy](m))
}

func TestChargeMeterFake(t *testing.T) {
	m := newChargeMeter(struct{}{})
	require.NotNil(t, m.fake)

	m.fake.SetPower(42)
	f, err := m.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 42.0, f)

	assert.False(t, api.HasCap[api.MeterEnergy](m))
}

package api

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mock decorated type with capability registry
type decoratedMeter struct {
	Meter
	caps map[reflect.Type]any
}

func (d *decoratedMeter) Capability(typ reflect.Type) (any, bool) {
	c, ok := d.caps[typ]
	return c, ok
}

type testMeterImportImpl struct{}

func (t *testMeterImportImpl) ImportEnergy() (float64, error) {
	return 99.0, nil
}

type testMeterImpl struct{}

func (t *testMeterImpl) CurrentPower() (float64, error) {
	return 100.0, nil
}

func TestCap_DirectTypeAssertion(t *testing.T) {
	// concrete type that directly implements MeterImport
	impl := &testMeterImportImpl{}

	me, ok := Cap[MeterImport](impl)
	require.True(t, ok)

	energy, err := me.ImportEnergy()
	assert.NoError(t, err)
	assert.Equal(t, 99.0, energy)
}

func TestCap_CapableRegistryLookup(t *testing.T) {
	base := &testMeterImpl{}

	decorated := &decoratedMeter{
		Meter: base,
		caps: map[reflect.Type]any{
			reflect.TypeFor[MeterImport](): &testMeterImportImpl{},
		},
	}

	// should find MeterImport via registry
	me, ok := Cap[MeterImport](decorated)
	require.True(t, ok)

	energy, err := me.ImportEnergy()
	assert.NoError(t, err)
	assert.Equal(t, 99.0, energy)

	// should NOT find PhaseCurrents (not registered)
	_, ok = Cap[PhaseCurrents](decorated)
	assert.False(t, ok)
}

// decoratedCharger simulates a real decorated charger where Meter is NOT
// directly embedded but only available through the capability registry.
type decoratedCharger struct {
	caps map[reflect.Type]any
}

func (d *decoratedCharger) Capability(typ reflect.Type) (any, bool) {
	c, ok := d.caps[typ]
	return c, ok
}

func TestCap_ExtractedCapabilityLosesRegistry(t *testing.T) {
	// Reproduces https://github.com/evcc-io/evcc/issues/28915
	// When a Meter is extracted from a decorated charger via Cap[Meter],
	// the extracted impl does NOT carry the Capable interface, so
	// subsequent Cap[MeterImport] on the extracted value fails.
	decorated := &decoratedCharger{
		caps: map[reflect.Type]any{
			reflect.TypeFor[Meter]():       &testMeterImpl{},
			reflect.TypeFor[MeterImport](): &testMeterImportImpl{},
		},
	}

	// extract Meter from decorated source (slow path: from caps registry)
	mt, ok := Cap[Meter](decorated)
	require.True(t, ok)

	// Bug: extracted meter cannot find MeterImport because it's a standalone impl
	_, ok = Cap[MeterImport](mt)
	assert.False(t, ok, "extracted meter should NOT have MeterImport capability")

	// Fix: wrapping extracted meter with source's Capable preserves registry
	type capableMeter struct {
		Meter
		Capable
	}
	wrapped := &capableMeter{Meter: mt, Capable: decorated}

	me, ok := Cap[MeterImport](wrapped)
	require.True(t, ok, "wrapped meter should find MeterImport via Capable")

	energy, err := me.ImportEnergy()
	assert.NoError(t, err)
	assert.Equal(t, 99.0, energy)
}

func TestCap_NilValue(t *testing.T) {
	_, ok := Cap[MeterImport](nil)
	assert.False(t, ok)
}

func TestCap_DirectTakesPrecedence(t *testing.T) {
	// type that both directly implements AND has registry
	type directAndCapable struct {
		testMeterImportImpl
		caps map[reflect.Type]any //nolint:unused
	}

	v := &directAndCapable{}

	me, ok := Cap[MeterImport](v)
	require.True(t, ok)

	energy, err := me.ImportEnergy()
	assert.NoError(t, err)
	assert.Equal(t, 99.0, energy)
}

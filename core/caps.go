package core

import (
	"reflect"

	"github.com/evcc-io/evcc/api"
)

// capableMeter wraps a meter with capability lookup from its source.
// This preserves capability checks (like MeterEnergy, PhaseCurrents, PhaseVoltages) when
// the meter was extracted from a decorated charger's capability registry.
type capableMeter struct {
	api.Meter
	source any
}

// Capability implements the api.Capable interface. It first consults the
// source's capability registry (for decorated capabilities), then falls back
// to a direct type assertion on the source so statically-implemented
// interfaces (e.g. PhaseCurrents on the DaheimLaden charger) remain
// discoverable through the wrapper (https://github.com/evcc-io/evcc/issues/29877).
func (m *capableMeter) Capability(typ reflect.Type) (any, bool) {
	if c, ok := m.source.(api.Capable); ok {
		if impl, ok := c.Capability(typ); ok {
			return impl, true
		}
	}
	if v := reflect.ValueOf(m.source); v.IsValid() && v.Type().Implements(typ) {
		return m.source, true
	}
	return nil, false
}

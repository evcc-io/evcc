package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/wrapper"
)

// chargeMeter is the loadpoint's charge meter. Capability lookups are delegated to the source
// (meter device or charger) so decorated and static capabilities stay discoverable (#28915, #29877).
type chargeMeter struct {
	api.Meter
	source any
	fake   *wrapper.ChargeMeter // non-nil when power is simulated from offered current
}

var _ api.Delegator = (*chargeMeter)(nil)

// newChargeMeter creates the charge meter from a meter device or charger.
// A simulated meter is used if the source does not provide api.Meter.
func newChargeMeter(source any) *chargeMeter {
	m := &chargeMeter{source: source}

	if m.Meter, _ = api.Cap[api.Meter](source); m.Meter == nil {
		m.fake = new(wrapper.ChargeMeter)
		m.Meter = m.fake
	}

	return m
}

// Delegate implements the api.Delegator interface
func (m *chargeMeter) Delegate() any {
	return m.source
}

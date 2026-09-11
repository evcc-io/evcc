package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/core/wrapper"
)

// chargeMeter is the loadpoint's charge meter with capabilities resolved once at boot.
// Resolving against the source (meter device or charger) instead of the extracted api.Meter
// keeps decorated and statically implemented capabilities discoverable through api.Cap
// (https://github.com/evcc-io/evcc/issues/28915, https://github.com/evcc-io/evcc/issues/29877).
type chargeMeter struct {
	api.Meter
	implement.Caps
	fake *wrapper.ChargeMeter // non-nil when power is simulated from offered current
}

// newChargeMeter creates the charge meter from a meter device or charger.
// A simulated meter is used if the source does not provide api.Meter.
func newChargeMeter(source any) *chargeMeter {
	m := &chargeMeter{Caps: implement.New()}

	if m.Meter, _ = api.Cap[api.Meter](source); m.Meter == nil {
		m.fake = new(wrapper.ChargeMeter)
		m.Meter = m.fake
		return m
	}

	copyCap[api.MeterEnergy](m.Caps, source)
	copyCap[api.MeterReturnEnergy](m.Caps, source)
	copyCap[api.PhaseCurrents](m.Caps, source)
	copyCap[api.PhaseVoltages](m.Caps, source)

	return m
}

// copyCap registers capability T of source on c if present
func copyCap[T any](c implement.Caps, source any) {
	v, _ := api.Cap[T](source)
	implement.May(c, v)
}

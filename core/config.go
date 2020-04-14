package core

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core/wrapper"
	"github.com/andig/evcc/provider"
)

// NewMeterFromConfig creates api.Meter from config
func NewMeterFromConfig(log *api.Logger, other map[string]interface{}) api.Meter {
	cc := struct {
		Power  provider.Config
		Energy *provider.Config // optional
	}{}
	api.DecodeOther(log, other, &cc)

	m := NewMeter(provider.NewFloatGetterFromConfig(log, cc.Power))

	// decorate Meter with MeterEnergy
	if cc.Energy != nil {
		m = &wrapper.CompositeMeter{
			Meter:       m,
			MeterEnergy: NewMeterEnergy(provider.NewFloatGetterFromConfig(log, *cc.Energy)),
		}
	}

	return m
}

package core

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core/wrapper"
	"github.com/andig/evcc/provider"
)

// MeterConfig is the generic meter configuration
type MeterConfig struct {
	Name   string
	Power  provider.Config
	Energy *provider.Config // optional
}

// NewMeterFromConfig creates api.Meter from config
func NewMeterFromConfig(log *api.Logger, mc MeterConfig) api.Meter {
	m := NewMeter(provider.NewFloatGetterFromConfig(log, mc.Power))

	// decorate Meter with MeterEnergy
	if mc.Energy != nil {
		m = &wrapper.CompositeMeter{
			Meter:       m,
			MeterEnergy: NewMeterEnergy(provider.NewFloatGetterFromConfig(log, *mc.Energy)),
		}
	}

	return m
}

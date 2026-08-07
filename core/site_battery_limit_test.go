package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
)

type mockBatteryPowerLimiter struct {
	api.Meter
	charge, discharge float64
}

func (m *mockBatteryPowerLimiter) GetPowerLimits() (float64, float64) {
	return m.charge, m.discharge
}

type mockMeter struct {
	api.Meter
}

func (m *mockMeter) CurrentPower() (float64, error) {
	return 0, nil
}

func TestBatteryMaxDischargePowerAggregation(t *testing.T) {
	site := &Site{
		log: util.NewLogger("foo"),
	}

	// one battery with limit, one without
	m1 := &mockBatteryPowerLimiter{Meter: &mockMeter{}, discharge: 2000}
	m2 := &mockMeter{}

	site.batteryMeters = []config.Device[api.Meter]{
		config.NewStaticDevice[api.Meter](config.Named{Name: "bat1"}, m1),
		config.NewStaticDevice[api.Meter](config.Named{Name: "bat2"}, m2),
	}

	site.updateBatteryMeters()
	assert.Equal(t, 0.0, site.GetBatteryMaxDischargePower())

	// both batteries with limit
	m3 := &mockBatteryPowerLimiter{Meter: &mockMeter{}, discharge: 3000}
	site.batteryMeters = []config.Device[api.Meter]{
		config.NewStaticDevice[api.Meter](config.Named{Name: "bat1"}, m1),
		config.NewStaticDevice[api.Meter](config.Named{Name: "bat3"}, m3),
	}

	site.updateBatteryMeters()
	assert.Equal(t, 5000.0, site.GetBatteryMaxDischargePower())
}

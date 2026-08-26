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

func (m *mockBatteryPowerLimiter) Soc() (float64, error) {
	return 50, nil
}

type mockMeter struct {
	api.Meter
}

func (m *mockMeter) CurrentPower() (float64, error) {
	return 0, nil
}

func (m *mockMeter) Soc() (float64, error) {
	return 50, nil
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
	assert.Nil(t, site.GetBatteryMaxDischargePower())

	// both batteries with limit
	m3 := &mockBatteryPowerLimiter{Meter: &mockMeter{}, discharge: 3000}
	site.batteryMeters = []config.Device[api.Meter]{
		config.NewStaticDevice[api.Meter](config.Named{Name: "bat1"}, m1),
		config.NewStaticDevice[api.Meter](config.Named{Name: "bat3"}, m3),
	}

	site.updateBatteryMeters()
	if res := site.GetBatteryMaxDischargePower(); assert.NotNil(t, res) {
		assert.Equal(t, 5000.0, *res)
	}
}

type mockBatterySocLimiter struct {
	api.Meter
	soc, min, max float64
}

func (m *mockBatterySocLimiter) Soc() (float64, error) {
	return m.soc, nil
}

func (m *mockBatterySocLimiter) GetSocLimits() (float64, float64) {
	return m.min, m.max
}

type mockLimiter struct {
	mockBatterySocLimiter
	discharge float64
}

func (m *mockLimiter) GetPowerLimits() (float64, float64) {
	return 0, m.discharge
}

func TestBatteryMaxDischargePowerWithMinSoc(t *testing.T) {
	site := &Site{
		log: util.NewLogger("foo"),
	}

	// one battery empty (soc <= min), one normal
	m1 := &mockLimiter{mockBatterySocLimiter: mockBatterySocLimiter{Meter: &mockMeter{}, soc: 10, min: 20}, discharge: 2000}
	m2 := &mockLimiter{mockBatterySocLimiter: mockBatterySocLimiter{Meter: &mockMeter{}, soc: 50, min: 20}, discharge: 3000}

	site.batteryMeters = []config.Device[api.Meter]{
		config.NewStaticDevice[api.Meter](config.Named{Name: "bat1"}, m1),
		config.NewStaticDevice[api.Meter](config.Named{Name: "bat2"}, m2),
	}

	site.updateBatteryMeters()
	// Only m2 should contribute
	if res := site.GetBatteryMaxDischargePower(); assert.NotNil(t, res) {
		assert.Equal(t, 3000.0, *res)
	}

	// Both empty
	m3 := &mockLimiter{mockBatterySocLimiter: mockBatterySocLimiter{Meter: &mockMeter{}, soc: 15, min: 20}, discharge: 3000}
	site.batteryMeters = []config.Device[api.Meter]{
		config.NewStaticDevice[api.Meter](config.Named{Name: "bat1"}, m1),
		config.NewStaticDevice[api.Meter](config.Named{Name: "bat3"}, m3),
	}

	site.updateBatteryMeters()
	if res := site.GetBatteryMaxDischargePower(); assert.NotNil(t, res) {
		assert.Equal(t, 0.0, *res)
	}
}

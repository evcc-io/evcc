package core

import (
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
)

type unavailableSocBattery struct{ mockBatteryPowerLimiter }

func (m *unavailableSocBattery) Soc() (float64, error) {
	return 0, errors.New("battery offline")
}

func TestBatteryDischargeBudgetRequiresFreshSoc(t *testing.T) {
	s := &Site{log: util.NewLogger("test")}
	s.battery.Soc = 70 // cached display value from the previous successful read
	m := &unavailableSocBattery{mockBatteryPowerLimiter{Meter: &mockMeter{}, discharge: 3300}}
	s.batteryMeters = []config.Device[api.Meter]{config.NewStaticDevice[api.Meter](config.Named{Name: "battery"}, m)}
	s.updateBatteryMeters()
	assert.Equal(t, 70.0, s.GetBatterySoc(), "display keeps last known SoC")
	if cap := s.GetBatteryMaxDischargePower(); assert.NotNil(t, cap) {
		assert.Zero(t, *cap, "cached SoC cannot authorize battery support")
	}
}

type mockBatteryPowerLimiter struct {
	api.Meter
	charge, discharge float64
}

func (m *mockBatteryPowerLimiter) GetPowerLimits() (float64, float64, error) {
	return m.charge, m.discharge, nil
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

func (m *mockBatterySocLimiter) GetSocLimits() (float64, float64, error) {
	return m.min, m.max, nil
}

type mockLimiter struct {
	mockBatterySocLimiter
	discharge float64
}

func (m *mockLimiter) GetPowerLimits() (float64, float64, error) {
	return 0, m.discharge, nil
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
	if assert.Len(t, site.battery.Devices, 2) {
		assert.Equal(t, 20.0, *site.battery.Devices[0].MinSoc, "soc limits kept on the measurement")
	}
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

type modeBudgetBattery struct {
	mockBatteryPowerLimiter
	mode api.BatteryOperatingMode
}

func (m *modeBudgetBattery) BatteryControlState() (api.BatteryControlState, error) {
	return api.BatteryControlState{Mode: m.mode, ObservedAt: time.Now()}, nil
}
func TestBatteryDischargeBudgetRequiresAutomaticMode(t *testing.T) {
	m := &modeBudgetBattery{mockBatteryPowerLimiter: mockBatteryPowerLimiter{Meter: &mockMeter{}, discharge: 3300}}
	s := &Site{log: util.NewLogger("test")}
	s.batteryMeters = []config.Device[api.Meter]{config.NewStaticDevice[api.Meter](config.Named{Name: "battery"}, m)}
	for _, mode := range []api.BatteryOperatingMode{api.BatteryOperatingManual, api.BatteryOperatingUnknown, api.BatteryOperatingAuto} {
		m.mode = mode
		s.updateBatteryMeters()
		cap := s.GetBatteryMaxDischargePower()
		if assert.NotNil(t, cap) {
			expected := 0.0
			if mode == api.BatteryOperatingAuto {
				expected = 3300
			}
			assert.Equal(t, expected, *cap)
		}
	}
}

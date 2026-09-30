package core

import (
	"math"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type humidityTestCharger struct {
	api.Charger
	humidity float64
	err      error
	calls    int
	features []api.Feature
}

func (c *humidityTestCharger) CurrentHumidity() (float64, error) {
	c.calls++
	return c.humidity, c.err
}

func (c *humidityTestCharger) Features() []api.Feature {
	return c.features
}

func TestEvaluateDehumidifierState(t *testing.T) {
	for _, tc := range []struct {
		name        string
		mode        api.ChargeMode
		enabled     bool
		humidity    float64
		hysteresis  float64
		minOn       time.Duration
		minOff      time.Duration
		elapsed     time.Duration
		pvAvailable bool
		want        bool
	}{
		{name: "start above upper threshold", mode: api.ModeSmart, humidity: 52.1, hysteresis: 2, pvAvailable: true, want: true},
		{name: "hold at upper threshold", mode: api.ModeSmart, humidity: 52, hysteresis: 2, pvAvailable: true, want: false},
		{name: "stop below lower threshold", mode: api.ModeSmart, enabled: true, humidity: 47.9, hysteresis: 2, pvAvailable: true, want: false},
		{name: "hold at lower threshold", mode: api.ModeSmart, enabled: true, humidity: 48, hysteresis: 2, pvAvailable: true, want: true},
		{name: "hold enabled in deadband", mode: api.ModeSmart, enabled: true, humidity: 50, hysteresis: 2, pvAvailable: true, want: true},
		{name: "hold disabled in deadband", mode: api.ModeSmart, humidity: 50, hysteresis: 2, pvAvailable: true, want: false},
		{name: "custom hysteresis threshold", mode: api.ModeSmart, humidity: 54.5, hysteresis: 4.5, pvAvailable: true, want: false},
		{name: "custom hysteresis starts beyond threshold", mode: api.ModeSmart, humidity: 54.6, hysteresis: 4.5, pvAvailable: true, want: true},
		{name: "smart mode waits for PV", mode: api.ModeSmart, humidity: 60, hysteresis: 2, want: false},
		{name: "minimum on overrides unavailable PV", mode: api.ModeSmart, enabled: true, humidity: 40, hysteresis: 2, minOn: 10 * time.Minute, elapsed: 9 * time.Minute, want: true},
		{name: "minimum on time expired", mode: api.ModeSmart, enabled: true, humidity: 40, hysteresis: 2, pvAvailable: true, minOn: 10 * time.Minute, elapsed: 10 * time.Minute, want: false},
		{name: "minimum off time overrides available PV", mode: api.ModeSmart, humidity: 60, hysteresis: 2, minOff: 5 * time.Minute, elapsed: 4 * time.Minute, want: false},
		{name: "minimum off time expired", mode: api.ModeSmart, humidity: 60, hysteresis: 2, pvAvailable: true, minOff: 5 * time.Minute, elapsed: 5 * time.Minute, want: true},
		{name: "now mode ignores PV", mode: api.ModeNow, humidity: 60, hysteresis: 2, want: true},
		{name: "off overrides minimum on time", mode: api.ModeOff, enabled: true, humidity: 60, hysteresis: 2, minOn: 10 * time.Minute, elapsed: time.Minute, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clk := clock.NewMock()
			now := clk.Now()
			lp := NewLoadpoint(util.NewLogger("test"), nil)
			lp.clock = clk
			lp.mode = tc.mode
			lp.enabled = tc.enabled
			lp.dehumidifierSwitched = now
			lp.Dehumidifier = loadpoint.DehumidifierConfig{
				TargetHumidity: 50,
				Hysteresis:     tc.hysteresis,
				MinOnTime:      tc.minOn,
				MinOffTime:     tc.minOff,
			}
			clk.Set(now.Add(tc.elapsed))

			assert.Equal(t, tc.want, lp.evaluateDehumidifierState(tc.humidity, tc.pvAvailable))
		})
	}
}

func TestDehumidifierConfigValidate(t *testing.T) {
	valid := loadpoint.DefaultDehumidifierConfig()
	assert.NoError(t, valid.Validate())

	for _, tc := range []struct {
		name   string
		change func(*loadpoint.DehumidifierConfig)
	}{
		{name: "target below range", change: func(c *loadpoint.DehumidifierConfig) { c.TargetHumidity = -1 }},
		{name: "target above range", change: func(c *loadpoint.DehumidifierConfig) { c.TargetHumidity = 101 }},
		{name: "non-finite target", change: func(c *loadpoint.DehumidifierConfig) { c.TargetHumidity = math.NaN() }},
		{name: "negative hysteresis", change: func(c *loadpoint.DehumidifierConfig) { c.Hysteresis = -1 }},
		{name: "non-finite hysteresis", change: func(c *loadpoint.DehumidifierConfig) { c.Hysteresis = math.Inf(1) }},
		{name: "negative guard duration", change: func(c *loadpoint.DehumidifierConfig) { c.MinOnTime = -time.Second }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := valid
			tc.change(&config)
			require.Error(t, config.Validate())
		})
	}
}

func TestUpdateDehumidifier(t *testing.T) {
	for _, tc := range []struct {
		name        string
		mode        api.ChargeMode
		enabled     bool
		humidity    float64
		sensorError error
		wantEnabled bool
		wantError   bool
	}{
		{name: "enables above threshold", mode: api.ModeNow, humidity: 53, wantEnabled: true},
		{name: "sensor error preserves state", mode: api.ModeNow, humidity: 53, sensorError: assert.AnError, wantEnabled: false, wantError: true},
		{name: "invalid value preserves state", mode: api.ModeNow, humidity: math.NaN(), wantEnabled: false, wantError: true},
		{name: "off mode disables despite sensor error", mode: api.ModeOff, enabled: true, humidity: 0, sensorError: assert.AnError, wantEnabled: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			base := api.NewMockCharger(ctrl)
			base.EXPECT().Enable(!tc.enabled).DoAndReturn(func(enabled bool) error {
				require.Equal(t, tc.wantEnabled, enabled)
				return nil
			}).Times(boolToInt(tc.enabled != tc.wantEnabled))
			charger := &humidityTestCharger{Charger: base, humidity: tc.humidity, err: tc.sensorError}

			lp := NewLoadpoint(util.NewLogger("test"), settings.NewMemorySettings())
			lp.charger = charger
			lp.mode = tc.mode
			lp.enabled = tc.enabled
			lp.dehumidifierSwitched = lp.clock.Now().Add(-time.Hour)
			lp.Dehumidifier.MinOnTime = 0
			lp.Dehumidifier.MinOffTime = 0
			lp.uiChan = make(chan util.Param, 8)

			err := lp.updateDehumidifier(0, 0, false, false)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantEnabled, lp.enabled)
			if tc.mode == api.ModeOff {
				assert.Zero(t, charger.calls)
			} else {
				assert.Equal(t, 1, charger.calls)
			}
		})
	}
}

func TestUpdateDehumidifierSmartPV(t *testing.T) {
	previousVoltage := Voltage
	Voltage = 230
	t.Cleanup(func() { Voltage = previousVoltage })

	for _, tc := range []struct {
		name      string
		sitePower float64
		wantOn    bool
	}{
		{name: "insufficient surplus", sitePower: 0, wantOn: false},
		{name: "surplus covers minimum load", sitePower: -1500, wantOn: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			base := api.NewMockCharger(ctrl)
			if tc.wantOn {
				base.EXPECT().Enable(true).Return(nil)
			}

			lp := NewLoadpoint(util.NewLogger("test"), settings.NewMemorySettings())
			lp.charger = &humidityTestCharger{
				Charger:  base,
				humidity: 60,
				features: []api.Feature{api.IntegratedDevice, api.SwitchDevice, api.Dehumidifier},
			}
			lp.mode = api.ModeSmart
			lp.phases = 1
			lp.Enable.Delay = 0
			lp.Dehumidifier.MinOnTime = 0
			lp.Dehumidifier.MinOffTime = 0
			lp.dehumidifierSwitched = lp.clock.Now().Add(-time.Hour)
			lp.uiChan = make(chan util.Param, 8)

			require.NoError(t, lp.updateDehumidifier(tc.sitePower, 0, false, false))
			assert.Equal(t, tc.wantOn, lp.enabled)
		})
	}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

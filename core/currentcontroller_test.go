package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func testControllerLoadpoint(charger api.Charger, phasesConfigured, phases int) *Loadpoint {
	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.wakeUpTimer = NewTimer()
	currentController(lp).minCurrent = minA
	currentController(lp).maxCurrent = maxA
	currentController(lp).phasesConfigured = phasesConfigured
	currentController(lp).phases = phases
	lp.charger = charger
	lp.status = api.StatusC
	return lp
}

func TestControllerEnvelope(t *testing.T) {
	Voltage = 230

	tc := []struct {
		desc                      string
		switchable                bool
		phasesConfigured, phases  int
		activeMin, reachable, max float64
	}{
		{"auto switching, 3p active", true, 0, 3, 3 * minA * Voltage, minA * Voltage, 3 * maxA * Voltage},
		{"auto switching, 1p active", true, 0, 1, minA * Voltage, minA * Voltage, 3 * maxA * Voltage},
		{"locked 3p", true, 3, 3, 3 * minA * Voltage, 3 * minA * Voltage, 3 * maxA * Voltage},
		{"locked 1p", true, 1, 1, minA * Voltage, minA * Voltage, maxA * Voltage},
		{"fixed 3p, no switching", false, 3, 3, 3 * minA * Voltage, 3 * minA * Voltage, 3 * maxA * Voltage},
	}

	for _, tc := range tc {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			var charger api.Charger = api.NewMockCharger(ctrl)
			if tc.switchable {
				charger = struct {
					*api.MockCharger
					*api.MockPhaseSwitcher
				}{api.NewMockCharger(ctrl), api.NewMockPhaseSwitcher(ctrl)}
			}

			lp := testControllerLoadpoint(charger, tc.phasesConfigured, tc.phases)
			c := currentController(lp)

			assert.Equal(t, tc.activeMin, c.activeMinPower(), "active min power")
			assert.Equal(t, tc.reachable, c.reachableMinPower(), "reachable min power")
			assert.Equal(t, tc.max, c.effectiveMaxPower(), "effective max power")

			// api.PowerController envelope
			assert.Equal(t, tc.reachable, c.MinPower(), "envelope min power")
			assert.Equal(t, tc.max, c.MaxPower(), "envelope max power")
		})
	}
}

func TestControllerSetPowerDisable(t *testing.T) {
	ctrl := gomock.NewController(t)
	Voltage = 230

	charger := api.NewMockCharger(ctrl)
	charger.EXPECT().Enable(false).Return(nil)

	lp := testControllerLoadpoint(charger, 3, 3)
	currentController(lp).enabled = true
	currentController(lp).offeredCurrent = minA

	require.NoError(t, currentController(lp).SetPower(0))
	assert.False(t, currentController(lp).enabled, "charger must be disabled")
}

func TestControllerSetPowerClampsToEnvelope(t *testing.T) {
	Voltage = 230

	tc := []struct {
		desc     string
		power    float64
		expected int64
	}{
		// a positive setpoint expresses that charging shall happen:
		// sub-minimum targets are clamped up to the minimum
		{"sub-min clamps up", minA * Voltage, int64(minA)}, // 1p min at 3p active
		{"interior converts to current", 10 * 3 * Voltage, 10},
		{"above max clamps down", (maxA + 5) * 3 * Voltage, int64(maxA)},
	}

	for _, tc := range tc {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			charger := api.NewMockCharger(ctrl)
			charger.EXPECT().Enable(true).Return(nil)
			charger.EXPECT().MaxCurrent(tc.expected).Return(nil)

			lp := testControllerLoadpoint(charger, 3, 3)

			// surplus context: keep phase hysteresis out of the way
			surplus := 0.0
			currentController(lp).Prepare(surplus)

			require.NoError(t, currentController(lp).SetPower(tc.power))
			assert.Equal(t, float64(tc.expected), currentController(lp).offeredCurrent)
		})
	}
}

func TestControllerSetPowerFastCharging(t *testing.T) {
	ctrl := gomock.NewController(t)
	Voltage = 230

	plainCharger := api.NewMockCharger(ctrl)
	plainCharger.EXPECT().Enable(true).Return(nil)
	plainCharger.EXPECT().MaxCurrent(int64(maxA)).Return(nil)

	switcher := api.NewMockPhaseSwitcher(ctrl)
	switcher.EXPECT().Phases1p3p(3).Return(nil)

	charger := struct {
		*api.MockCharger
		*api.MockPhaseSwitcher
	}{plainCharger, switcher}

	lp := testControllerLoadpoint(charger, 0, 1)
	c := currentController(lp)

	// full envelope target scales up immediately and sets maximum current
	require.NoError(t, c.SetPower(c.effectiveMaxPower()))
	assert.Equal(t, 3, c.phases, "expected immediate scale up")
	assert.Equal(t, float64(maxA), c.offeredCurrent)
}

func TestControllerSurplusConsumedPerCycle(t *testing.T) {
	ctrl := gomock.NewController(t)
	Voltage = 230

	charger := api.NewMockCharger(ctrl)
	charger.EXPECT().Enable(true).Return(nil)
	charger.EXPECT().MaxCurrent(int64(10)).Return(nil)

	lp := testControllerLoadpoint(charger, 3, 3)

	surplus := -1000.0
	currentController(lp).Prepare(surplus)

	require.NoError(t, currentController(lp).SetPower(10*3*Voltage))
	assert.Nil(t, currentController(lp).surplus, "surplus must be consumed")
}

type fakePowerCharger struct {
	api.Charger
	power float64
}

func (c *fakePowerCharger) SetPower(power float64) error {
	c.power = power
	return nil
}

// a natively power-controlled charger receives the setpoint in W instead of a current write
func TestControllerNativePowerCharger(t *testing.T) {
	ctrl := gomock.NewController(t)
	Voltage = 230

	charger := &fakePowerCharger{Charger: api.NewMockCharger(ctrl)}
	charger.Charger.(*api.MockCharger).EXPECT().Enable(true).Return(nil)

	lp := testControllerLoadpoint(charger, 1, 1)
	c := currentController(lp)

	require.NoError(t, c.SetPower(10*Voltage))
	assert.Equal(t, 10*Voltage, charger.power, "expected watt write")
	assert.Equal(t, 10.0, c.offeredCurrent)
}

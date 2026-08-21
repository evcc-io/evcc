package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

type mockSite struct {
	site.API
	maxDischargePower *float64
	batterySoc        float64
	residualPower     float64
	optimized         int
}

func (m *mockSite) Optimize() {
	m.optimized++
}

func (m *mockSite) GetBatteryMaxDischargePower() *float64 {
	return m.maxDischargePower
}

func (m *mockSite) GetBatterySoc() float64 {
	return m.batterySoc
}

func (m *mockSite) GetResidualPower() float64 {
	return m.residualPower
}

type phaseSwitchCharger struct {
	api.Charger
}

func (phaseSwitchCharger) Phases1p3p(int) error { return nil }

func TestBoostActive(t *testing.T) {
	s := &mockSite{}
	lp := &Loadpoint{
		log:  util.NewLogger("lp"),
		site: s,
	}

	// test boost disabled and soc is too low
	lp.batteryBoost = false
	lp.batteryBoostLimit = 50
	s.batterySoc = 49
	assert.False(t, lp.IsBatteryBoostActive(), "disabled & soc < limit")

	// test boost disabled but soc would be high enough
	s.batterySoc = 51
	assert.False(t, lp.IsBatteryBoostActive(), "disabled")

	// test boost enabled but soc is too low
	lp.batteryBoost = true
	s.batterySoc = 49
	assert.False(t, lp.IsBatteryBoostActive(), "soc < limit")

	// test boost enabled and soc is high enough
	s.batterySoc = 51
	assert.True(t, lp.IsBatteryBoostActive(), "enabled & soc > limit")

	// test if boost limit 0 always prioritizes the loadpoint over the battery
	lp.batteryBoostLimit = 0
	s.batterySoc = 0
	assert.True(t, lp.IsBatteryBoostActive(), "soc = limit = 0")

	// test if boost limit 100 never activates
	lp.batteryBoostLimit = 100
	s.batterySoc = 100
	assert.False(t, lp.IsBatteryBoostActive(), "soc = limit = 100")
}

func TestBatteryBoost(t *testing.T) {
	Voltage = 230
	s := &mockSite{}
	lp := &Loadpoint{
		log:        util.NewLogger("lp"),
		site:       s,
		charger:    phaseSwitchCharger{},
		minCurrent: 6,
		maxCurrent: 16,
		phases:     1,
	}

	lp.batteryBoostLimit = 50
	s.batterySoc = 51

	// tests with maxDischargePower = nil
	s.maxDischargePower = nil

	// test boost disabled:
	// - boostPower() returns the unmodified thresholds
	// - since the enable threshold is set to 0
	//   it defaults to the negated assumed min charge power
	lp.batteryBoost = false
	batteryPower := 1337.0
	boost := lp.boostPower(batteryPower)
	enable, disable := lp.boostThresholds(batteryPower)
	assert.Equal(t, 0.0, boost)
	assert.Equal(t, -1380.0, enable)
	assert.Equal(t, 0.0, disable)

	// test enabled but soc < limit:
	// - results are equal to the boost disabled case
	lp.batteryBoost = true
	s.batterySoc = 49
	boost = lp.boostPower(batteryPower)
	enable, disable = lp.boostThresholds(batteryPower)
	assert.Equal(t, 0.0, boost)
	assert.Equal(t, -1380.0, enable)
	assert.Equal(t, 0.0, disable)
	s.batterySoc = 51

	// test battery charging:
	// - boostPower() ignores battery charge power
	//   and returns at least
	//   EffectiveStepPower + max(0, residualPower+100)
	//   if maxDischargePower = nil
	batteryPower = -1
	boost = lp.boostPower(batteryPower)
	assert.Equal(t, 330.0, boost)

	// test battery power = 0:
	// - enable  = -230 -    0   = -230
	// - disable = -230 + 1380   = 1150
	// - gap     =  230*(6*3-16) =  460
	//
	// - assume lp is disabled:
	//     sitePower = 0 - 330 = -330 <= -230 (lp enables)
	batteryPower = 0
	boost = lp.boostPower(batteryPower)
	gap := lp.boostPhaseScaling()
	enable, disable = lp.boostThresholds(batteryPower)
	assert.Equal(t, 330.0, boost)
	assert.Equal(t, 460.0, gap)
	assert.Equal(t, -230.0, enable)
	assert.Equal(t, 1150.0, disable)

	// test battery discharging with 100W:
	// - enable  = -230 -  100 = -330
	// - disable = -230 + 1380 = 1150
	//
	// - assume lp is disabled:
	//     sitePower = 100 - 430 = -330 <= -330 (lp enables)
	batteryPower = 100
	boost = lp.boostPower(batteryPower)
	enable, disable = lp.boostThresholds(batteryPower)
	assert.Equal(t, 430.0, boost)
	assert.Equal(t, -330.0, enable)
	assert.Equal(t, 1150.0, disable)

	// test battery discharging with 200W:
	// - enable  = -230 -  200 = -430
	// - disable = -230 + 1380 = 1150
	//
	// - assume lp is disabled:
	//     sitePower = 200 - 530 = -330 > -430 (lp stays disabled)
	//
	// - assume lp is enabled, 1380W charge power:
	//   - assume 1180W grid import + 200W from battery:
	//       sitePower = 1380 - 530 = 850 < 1150 (lp stays enabled)
	//   - assume 1480W grid import (100W grid + 200W battery non-lp load):
	//       sitePower = 1680 - 530 = 1150 >= 1150 (lp disables)
	batteryPower = 200
	boost = lp.boostPower(batteryPower)
	enable, disable = lp.boostThresholds(batteryPower)
	assert.Equal(t, 530.0, boost)
	assert.Equal(t, -430.0, enable)
	assert.Equal(t, 1150.0, disable)

	// test manual thresholds:
	// - we cannot safely assume that a manual enable threshold
	//   is intended to affect the boost enable threshold.
	//   Thats why the enable threshold setting will only
	//   affect the resulting boost disable threshold.
	//   The disable threshold setting is used to shift both
	//   boost thresholds, so a negative disable threshold shifts
	//   the resulting boost enable threshold towards (more) battery charge power.
	//   With this logic users are free to manipulate both boost thresholds.
	lp.Enable.Threshold = -9000
	lp.Disable.Threshold = 1
	enable, disable = lp.boostThresholds(batteryPower)
	assert.Equal(t, -429.0, enable)
	assert.Equal(t, 8772.0, disable)

	lp.Disable.Threshold = -1
	enable, disable = lp.boostThresholds(batteryPower)
	assert.Equal(t, -431.0, enable)
	assert.Equal(t, 8768.0, disable)

	// test low max current (= big power gap):
	// - gap = 230*(6*3-6) = 2760W
	lp.maxCurrent = 6
	gap = lp.boostPhaseScaling()
	assert.Equal(t, 2760.0, gap)
	lp.maxCurrent = 16

	// tests with maxDischargePower != nil:
	// - the battery power parameter is irrelevant
	//   if maxDischargePower != nil
	s.maxDischargePower = new(0.0)
	lp.Enable.Threshold = 0
	lp.Disable.Threshold = 0

	// test maxDischargePower = 0:
	// - EffectiveStepPower is the minimum to avoid charging
	// - enable  = -min(230, 1380)   = -230
	// - disable = -230 + 1380       = 1150
	// - gap     =  230*(6*3-16)-230 =  230
	//   (subtract max(maxDischargePower, EffectiveStepPower)=230)
	batteryPower = -4711
	boost = lp.boostPower(batteryPower)
	gap = lp.boostPhaseScaling()
	enable, disable = lp.boostThresholds(batteryPower)
	assert.Equal(t, 230.0, boost)
	assert.Equal(t, 230.0, gap)
	assert.Equal(t, -230.0, enable)
	assert.Equal(t, 1150.0, disable)

	// test 230 < maxDischargePower < 1380:
	// - EffectiveStepPower is irrelevant,
	//   battery can be kept discharging without it
	// - enable  = -min(231, 1380)   = -231
	// - disable = -231 + 1380       = 1149
	// - gap     =  230*(6*3-16)-231 =  229
	s.maxDischargePower = new(231.0)
	batteryPower = 4711
	boost = lp.boostPower(batteryPower)
	gap = lp.boostPhaseScaling()
	enable, disable = lp.boostThresholds(batteryPower)
	assert.Equal(t, 231.0, boost)
	assert.Equal(t, 229.0, gap)
	assert.Equal(t, -231.0, enable)
	assert.Equal(t, 1149.0, disable)

	// test maxDischargePower >= 1380:
	// - enable  = -min(5000, 1380) = -1380
	// - disable = -1380 + 1380     =     0
	s.maxDischargePower = new(5000.0)
	batteryPower = 0
	boost = lp.boostPower(batteryPower)
	gap = lp.boostPhaseScaling()
	enable, disable = lp.boostThresholds(batteryPower)
	assert.Equal(t, 5000.0, boost)
	assert.Equal(t, 0.0, gap)
	assert.Equal(t, -1380.0, enable)
	assert.Equal(t, 0.0, disable)

	// test manual thresholds:
	lp.Enable.Threshold = -9000
	lp.Disable.Threshold = 1
	enable, disable = lp.boostThresholds(0)
	assert.Equal(t, -4999.0, enable)
	assert.Equal(t, 4002.0, disable)

	lp.Disable.Threshold = -1
	enable, disable = lp.boostThresholds(-333)
	assert.Equal(t, -5001.0, enable)
	assert.Equal(t, 3998.0, disable)

	s.maxDischargePower = new(10000.0)
	enable, disable = lp.boostThresholds(0)
	assert.Equal(t, -9000.0, enable)
	assert.Equal(t, -1.0, disable)

	lp.Disable.Threshold = 1
	enable, disable = lp.boostThresholds(333)
	assert.Equal(t, -9000.0, enable)
	assert.Equal(t, 1.0, disable)

	// test low max current (= big power gap):
	// - gap = 230*(6*3-6) - 230 = 2530
	//   (subtract 230W coarse current adjustment)
	s.maxDischargePower = new(0.0)
	lp.maxCurrent = 6
	gap = lp.boostPhaseScaling()
	assert.Equal(t, 2530.0, gap)

	// test phases = 3:
	// - continue using 1p coarse current adjustment
	//   If we would subtract 690W now,
	//   we would have a negative 1p/3p switching hysteresis
	lp.phases = 3
	gap = lp.boostPhaseScaling()
	assert.Equal(t, 2530.0, gap)
}

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
	maxDischargePower float64
	residualPower     float64
}

func (m *mockSite) GetBatteryMaxDischargePower() float64 {
	return m.maxDischargePower
}

func (m *mockSite) GetResidualPower() float64 {
	return m.residualPower
}

func TestBoostPower(t *testing.T) {
	Voltage = 230
	lp := &Loadpoint{
		log:          util.NewLogger("lp"),
		status:       api.StatusC,
		batteryBoost: boostStart,
		maxCurrent:   16,
		phases:       3,
	}
	s := &mockSite{}
	lp.site = s

	// No max discharge power limit
	s.maxDischargePower = 0
	// EffectiveMaxPower will be 230 * 16 * 3 = 11040
	delta := lp.boostPower(0)
	assert.Equal(t, 11040.0, delta)
	assert.Equal(t, boostContinue, lp.batteryBoost)

	// With max discharge power limit
	s.maxDischargePower = 5000
	lp.batteryBoost = boostStart
	delta = lp.boostPower(0)
	assert.Equal(t, 5000.0, delta)
	assert.Equal(t, boostContinue, lp.batteryBoost)

	// boostContinue with limit
	lp.batteryBoost = boostContinue
	s.residualPower = 0
	// delta = math.Max(100, 0) = 100
	// plus EffectiveStepPower = 690
	// delta = 790
	// delta = min(790, max(0, 5000 - 0)) = 790
	// res = 0 + 790 + 0 = 790
	delta = lp.boostPower(0)
	assert.Equal(t, 790.0, delta)

	// boostContinue at limit
	// delta = min(790, max(0, 5000 - 5000)) = 0
	// res = 5000 + 0 + 0 = 5000
	delta = lp.boostPower(5000)
	assert.Equal(t, 5000.0, delta)

	// boostContinue over limit
	// delta = min(790, max(0, 5000 - 6000)) = 0
	// res = 6000 + 0 + 0 = 6000
	delta = lp.boostPower(6000)
	assert.Equal(t, 6000.0, delta)

	// boostStart while battery is charging (negative power)
	// battery charging at 2000W, limit is 5000W
	// max discharge capacity = 5000 - (-2000) = 7000W
	// res = max(0, -2000) + 7000 + 0 = 7000W
	lp.batteryBoost = boostStart
	delta = lp.boostPower(-2000)
	assert.Equal(t, 7000.0, delta)

	// boostContinue while battery is charging (negative power)
	// limit is 50W (less than the standard 790W delta)
	// without raw negative power, delta would be restricted to 50W
	// with raw negative power (-2000W), headroom is 2050W, so delta is allowed to be 790W
	s.maxDischargePower = 50
	s.residualPower = 0 // base delta = 100 + 690 = 790
	lp.batteryBoost = boostContinue
	delta = lp.boostPower(-2000)
	// res = max(0, -2000) + 790 + 0 = 790W
	assert.Equal(t, 790.0, delta)
}

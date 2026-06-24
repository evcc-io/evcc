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
}

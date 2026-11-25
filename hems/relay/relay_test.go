package relay

import (
	"math"
	"testing"

	"github.com/evcc-io/evcc/hems/hems"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCurtailmentNotConfigured verifies that relay never makes a curtailment
// statement, and that an active relay dims to maxPower.
func TestCurtailmentNotConfigured(t *testing.T) {
	c := new(Relay)

	// relay never makes a curtailment statement
	assert.Nil(t, c.CurtailedPercent())
	assert.Nil(t, c.MaxProductionPower())
	assert.Nil(t, hems.Curtailed(c))

	// active relay dims to maxPower
	c.maxPower = 1e3
	require.NoError(t, c.setConsumptionLimit(c.maxPower))
	assert.Equal(t, new(1e3), c.MaxConsumptionPower())
	assert.Equal(t, new(true), hems.Dimmed(c))

	// curtailment statement unaffected by relay activity
	assert.Nil(t, c.CurtailedPercent())
	assert.Nil(t, hems.Curtailed(c))
}

func TestInvalidPowerLimit(t *testing.T) {
	for _, maxPower := range []float64{0, -4200, math.NaN(), math.Inf(1)} {
		_, err := NewRelay(nil, nil, nil, maxPower, 0)
		assert.Error(t, err)
	}
}

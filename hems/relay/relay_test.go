package relay

import (
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

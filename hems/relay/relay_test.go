package relay

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/hems/hems"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRelayDoesNotReadLimit verifies that construction does not read the limit
// source. Blocking or failing here would stall startup, see #31999.
func TestNewRelayDoesNotReadLimit(t *testing.T) {
	limit := func() (bool, error) { return false, api.ErrOutdated }

	c, err := NewRelay(nil, limit, nil, 1e3, time.Second)
	require.NoError(t, err)
	assert.Nil(t, c.MaxConsumptionPower())
}

// TestCurtailmentNotConfigured verifies that relay never makes a curtailment
// statement, and that an active relay dims to maxPower.
func TestCurtailmentNotConfigured(t *testing.T) {
	c := new(Relay)

	// relay never makes a curtailment statement
	assert.Nil(t, c.CurtailedPercent())
	assert.Nil(t, c.MaxProductionPower())
	assert.Nil(t, hems.Curtailed(c))

	// no statement before the relay has been read
	assert.Nil(t, c.MaxConsumptionPower())
	assert.Nil(t, hems.Dimmed(c))

	// inactive relay is read and not dimmed
	require.NoError(t, c.setConsumptionLimit(0))
	assert.Equal(t, new(0.0), c.MaxConsumptionPower())
	assert.Equal(t, new(false), hems.Dimmed(c))

	// active relay dims to maxPower
	c.maxPower = 1e3
	require.NoError(t, c.setConsumptionLimit(c.maxPower))
	assert.Equal(t, new(1e3), c.MaxConsumptionPower())
	assert.Equal(t, new(true), hems.Dimmed(c))

	// curtailment statement unaffected by relay activity
	assert.Nil(t, c.CurtailedPercent())
	assert.Nil(t, hems.Curtailed(c))
}

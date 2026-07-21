package fnn

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/hems"
	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubSite implements site.API for testing — only GetGridPower is exercised.
type stubSite struct {
	site.API
}

func (s *stubSite) GetGridPower() float64 { return 0 }

func boolG(v bool) func() (bool, error) {
	return func() (bool, error) { return v, nil }
}

// TestNewFnnDoesNotReadRelays verifies that construction does not read the
// relay sources. Blocking or failing here would stall startup, see #31999.
func TestNewFnnDoesNotReadRelays(t *testing.T) {
	relay := func() (bool, error) { return false, api.ErrOutdated }

	fnn, err := NewFnn(nil, 1e3, 1e3, relay, nil, nil, relay, 0)
	require.NoError(t, err)
	assert.Nil(t, fnn.MaxConsumptionPower())
	assert.Nil(t, fnn.CurtailedPercent())
	assert.Nil(t, fnn.MaxProductionPower())
}

// TestCurtailmentNotConfigured verifies that without W3 no curtailment
// statement is made, while dimming via W4 remains available.
func TestCurtailmentNotConfigured(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	fnn, err := NewFnn(&stubSite{}, 1e3, 1e3, nil, nil, nil, boolG(true), 0)
	require.NoError(t, err)

	assert.Nil(t, fnn.CurtailedPercent())
	assert.Nil(t, fnn.MaxProductionPower())
	assert.Nil(t, hems.Curtailed(fnn))

	// no statement before the relays have been read
	assert.Nil(t, fnn.MaxConsumptionPower())
	assert.Nil(t, hems.Dimmed(fnn))

	require.NoError(t, fnn.runDim())
	assert.NotNil(t, fnn.MaxConsumptionPower())
	assert.NotNil(t, hems.Dimmed(fnn))
}

// TestDimmingNotConfigured verifies that without W4 no dimming statement is
// made, while curtailment via W3 remains available.
func TestDimmingNotConfigured(t *testing.T) {
	fnn, err := NewFnn(&stubSite{}, 0, 1e3, boolG(false), nil, nil, nil, 0)
	require.NoError(t, err)

	assert.Nil(t, fnn.MaxConsumptionPower())
	assert.Nil(t, hems.Dimmed(fnn))

	// no statement before the relays have been read
	assert.Nil(t, fnn.CurtailedPercent())
	assert.Nil(t, hems.Curtailed(fnn))

	require.NoError(t, fnn.runCurtail())
	assert.NotNil(t, fnn.CurtailedPercent())
	assert.NotNil(t, hems.Curtailed(fnn))
}

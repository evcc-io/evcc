package fnn

import (
	"testing"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/hems/hems"
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

// TestCurtailmentNotConfigured verifies that without W3 no curtailment
// statement is made, while dimming via W4 remains available.
func TestCurtailmentNotConfigured(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	fnn, err := NewFnn(&stubSite{}, 1e3, 1e3, nil, nil, nil, boolG(true), 0)
	require.NoError(t, err)

	assert.Nil(t, fnn.CurtailedPercent())
	assert.Nil(t, fnn.MaxProductionPower())
	assert.Nil(t, hems.Curtailed(fnn))

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

	assert.NotNil(t, fnn.CurtailedPercent())
	assert.NotNil(t, hems.Curtailed(fnn))
}

// TestProductionNominalMaxMissing verifies that S1/S2 require generator power
// to convert their percentage into a watt limit, while W3 alone does not.
func TestProductionNominalMaxMissing(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	_, err := NewFnn(&stubSite{}, 0, 0, boolG(false), boolG(false), nil, nil, 0)
	assert.Error(t, err)

	_, err = NewFnn(&stubSite{}, 0, 0, boolG(false), nil, boolG(false), nil, 0)
	assert.Error(t, err)

	fnn, err := NewFnn(&stubSite{}, 0, 0, boolG(true), nil, nil, nil, 0)
	require.NoError(t, err)
	assert.Equal(t, new(0.0), fnn.MaxProductionPower())
}

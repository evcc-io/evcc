package fnn

import (
	"testing"

	"github.com/evcc-io/evcc/hems/hems"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolG(v bool) func() (bool, error) {
	return func() (bool, error) { return v, nil }
}

// TestCurtailmentNotConfigured verifies that without W3 no curtailment
// statement is made, while dimming via W4 remains available.
func TestCurtailmentNotConfigured(t *testing.T) {
	fnn, err := NewFnn(nil, 1e3, 1e3, nil, nil, nil, boolG(true), 0)
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
	fnn, err := NewFnn(nil, 0, 1e3, boolG(false), nil, nil, nil, 0)
	require.NoError(t, err)

	assert.Nil(t, fnn.MaxConsumptionPower())
	assert.Nil(t, hems.Dimmed(fnn))

	assert.NotNil(t, fnn.CurtailedPercent())
	assert.NotNil(t, hems.Curtailed(fnn))
}

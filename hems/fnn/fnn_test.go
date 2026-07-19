package fnn

import (
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/hems"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util"
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

func errG() func() (bool, error) {
	return func() (bool, error) { return false, errors.New("read error") }
}

// TestCurtailmentNotConfigured verifies that without W3 no curtailment
// statement is made, while dimming via W4 remains available.
func TestCurtailmentNotConfigured(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	fnn, err := NewFnn(&stubSite{}, 1e3, 1e3, nil, nil, nil, boolG(true), 0, 0, 0)
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
	fnn, err := NewFnn(&stubSite{}, 0, 1e3, boolG(false), nil, nil, nil, 0, 0, 0)
	require.NoError(t, err)

	assert.Nil(t, fnn.MaxConsumptionPower())
	assert.Nil(t, hems.Dimmed(fnn))

	assert.NotNil(t, fnn.CurtailedPercent())
	assert.NotNil(t, hems.Curtailed(fnn))
}

// TestDecodeConfig verifies that failsafe config keys are accepted and decoded.
func TestDecodeConfig(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	_ = util.NewLogger("fnn")

	other := map[string]any{
		"maxDimPower":                         4200,
		"failsafeConsumptionActivePowerLimit": 4200,
		"failsafeDurationMinimum":             "30m",
		"w4": map[string]any{
			"source": "const",
			"value":  false,
		},
	}

	f, err := NewFromConfig(t.Context(), other, &stubSite{})
	require.NoError(t, err)
	assert.Equal(t, 4200.0, f.failsafeConsumptionLimit)
	assert.Equal(t, 30*time.Minute, f.failsafeDurationMinimum)
}

// TestFailsafeActivatesOnReadError verifies that a W4 read error triggers failsafe
// mode and the configured failsafe consumption limit is applied.
func TestFailsafeActivatesOnReadError(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	const failsafeLimit = 4200.0
	fnn, err := NewFnn(&stubSite{}, failsafeLimit, 0, nil, nil, nil, errG(), 0, failsafeLimit, 0)
	require.NoError(t, err)

	// construction calls runDim once — failsafe should already be active
	assert.True(t, fnn.failsafeActive)
	require.NotNil(t, fnn.MaxConsumptionPower())
	assert.Equal(t, failsafeLimit, *fnn.MaxConsumptionPower())
}

// TestFailsafeExitsAfterDuration verifies that failsafe is released once
// failsafeDurationMinimum has elapsed and a successful read follows.
func TestFailsafeExitsAfterDuration(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	const failsafeLimit = 4200.0
	// duration of 0 means failsafe exits on next successful read
	fnn, err := NewFnn(&stubSite{}, failsafeLimit, 0, nil, nil, nil, errG(), 0, failsafeLimit, 0)
	require.NoError(t, err)
	assert.True(t, fnn.failsafeActive)

	// switch to successful read
	fnn.w4 = boolG(false)
	require.NoError(t, fnn.runDim())

	assert.False(t, fnn.failsafeActive)
	require.NotNil(t, fnn.MaxConsumptionPower())
	assert.Equal(t, 0.0, *fnn.MaxConsumptionPower())
}

// TestFailsafeRemainsActiveDuringDuration verifies that failsafe stays active
// while failsafeDurationMinimum has not yet elapsed, even when reads succeed.
func TestFailsafeRemainsActiveDuringDuration(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	const failsafeLimit = 4200.0
	fnn, err := NewFnn(&stubSite{}, failsafeLimit, 0, nil, nil, nil, errG(), 0, failsafeLimit, time.Hour)
	require.NoError(t, err)
	assert.True(t, fnn.failsafeActive)

	// switch to successful read, but duration not yet elapsed
	fnn.w4 = boolG(false)
	require.NoError(t, fnn.runDim())

	assert.True(t, fnn.failsafeActive)
	require.NotNil(t, fnn.MaxConsumptionPower())
	assert.Equal(t, failsafeLimit, *fnn.MaxConsumptionPower())
}

// TestFailsafeNotConfiguredPropagatesError verifies that without a configured
// failsafe limit, a W4 read error is returned to the caller unchanged.
func TestFailsafeNotConfiguredPropagatesError(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	// no failsafe limit: construction error because runDim is called in NewFnn
	_, err := NewFnn(&stubSite{}, 1e3, 0, nil, nil, nil, errG(), 0, 0, 0)
	assert.Error(t, err)
}

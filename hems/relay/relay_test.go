package relay

import (
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/hems"
	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSiteStub struct {
	site.API
}

func (s *testSiteStub) GetGridPower() float64 { return 0 }

func boolG(v bool) func() (bool, error) {
	return func() (bool, error) { return v, nil }
}

func errG() func() (bool, error) {
	return func() (bool, error) { return false, errors.New("read error") }
}

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

func TestFailsafeActivatesOnReadError(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	const failsafeLimit = 4200.0
	c, err := NewRelay(&testSiteStub{}, boolG(false), nil, 1e3, 0, failsafeLimit, 0)
	require.NoError(t, err)
	c.w1 = errG()
	require.NoError(t, c.run())

	assert.True(t, c.failsafeActive)
	require.NotNil(t, c.MaxConsumptionPower())
	assert.Equal(t, failsafeLimit, *c.MaxConsumptionPower())
}

func TestFailsafeExitsAfterDuration(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	const failsafeLimit = 4200.0
	c, err := NewRelay(&testSiteStub{}, boolG(false), nil, 1e3, 0, failsafeLimit, 0)
	require.NoError(t, err)
	c.w1 = errG()
	require.NoError(t, c.run())
	assert.True(t, c.failsafeActive)

	c.w1 = boolG(false)
	require.NoError(t, c.run())

	assert.False(t, c.failsafeActive)
	require.NotNil(t, c.MaxConsumptionPower())
	assert.Equal(t, 0.0, *c.MaxConsumptionPower())
}

func TestFailsafeRemainsActiveDuringDuration(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	const failsafeLimit = 4200.0
	c, err := NewRelay(&testSiteStub{}, boolG(false), nil, 1e3, 0, failsafeLimit, time.Hour)
	require.NoError(t, err)
	c.w1 = errG()
	require.NoError(t, c.run())
	assert.True(t, c.failsafeActive)

	c.w1 = boolG(false)
	require.NoError(t, c.run())

	assert.True(t, c.failsafeActive)
	require.NotNil(t, c.MaxConsumptionPower())
	assert.Equal(t, failsafeLimit, *c.MaxConsumptionPower())
}

func TestFailsafeNotConfiguredPropagatesError(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	c, err := NewRelay(&testSiteStub{}, boolG(false), nil, 1e3, 0, 0, 0)
	require.NoError(t, err)

	want := errors.New("limit read error")
	c.w1 = func() (bool, error) { return false, want }

	assert.ErrorIs(t, c.run(), want)
}

func TestFailsafeNegativeDurationRejected(t *testing.T) {
	_, err := NewRelay(&testSiteStub{}, boolG(false), nil, 1e3, 0, 0, -time.Second)
	assert.ErrorContains(t, err, "failsafe duration cannot be negative")
}

func TestDecodeRejectsNegativeFailsafeLimit(t *testing.T) {
	other := map[string]any{
		"maxPower":                            4200,
		"failsafeConsumptionActivePowerLimit": -1,
		"limit": map[string]any{
			"source": "const",
			"value":  false,
		},
	}

	_, err := NewFromConfig(t.Context(), other, &testSiteStub{})
	assert.ErrorContains(t, err, "failsafe consumption limit cannot be negative")
}

package relay

import (
	"testing"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/require"
)

// stubSite implements site.API for testing — only GetGridPower is exercised.
type stubSite struct {
	site.API
}

func (s *stubSite) GetGridPower() float64 { return 0 }

// TestRelayNoNilState verifies MaxConsumptionPower is always determinable (w1
// is mandatory) — NewRelay reads it once so the state is valid immediately.
func TestRelayNoNilState(t *testing.T) {
	off := func() (bool, error) { return false, nil }
	c, err := NewRelay(&stubSite{}, off, nil, 1000, 0)
	require.NoError(t, err)

	require.NotNil(t, c.MaxConsumptionPower())
	require.Equal(t, 0.0, *c.MaxConsumptionPower())
	require.Nil(t, c.MaxProductionPower()) // scaffolding only, always nil
}

// TestRelayEdgeTriggered verifies that applying a limit (passthrough) only
// happens on a genuine transition, not on every steady-state run().
func TestRelayEdgeTriggered(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	calls := 0
	c := &Relay{
		site:        &stubSite{},
		w1:          func() (bool, error) { return true, nil },
		passthrough: func(bool) error { calls++; return nil },
		maxPower:    1000,
	}

	require.NoError(t, c.run())
	require.NoError(t, c.run())
	require.NoError(t, c.run())

	require.Equal(t, 1, calls, "passthrough must fire once on the edge, not every tick")
	require.Equal(t, 1000.0, *c.MaxConsumptionPower())
}

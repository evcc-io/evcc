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
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	off := func() (bool, error) { return false, nil }
	c, err := NewRelay(&stubSite{}, off, nil, 1000, 0)
	require.NoError(t, err)

	require.NotNil(t, c.MaxConsumptionPower())
	require.Equal(t, 0.0, *c.MaxConsumptionPower())
	require.Nil(t, c.MaxProductionPower()) // scaffolding only, always nil
}

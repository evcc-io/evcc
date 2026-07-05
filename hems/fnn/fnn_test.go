package fnn

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

func TestFnn(t *testing.T) {
	// TODO add site
	fnn, err := NewFnn(nil, 1e3, 1e3, nil, nil, nil, func() (bool, error) {
		return true, nil
	}, 0)
	require.NoError(t, err)
	require.Nil(t, fnn.CurtailedPercent())
	// require.NoError(t, fnn.runDim())
	// require.Equal(t, new(true), fnn.Dimmed())
}

// TestFnnNoLimitContract verifies api.HEMS's "nil = limiting undefined" contract: nil
// before the first run, then a definite value (0 = no limit) from then on.
func TestFnnNoLimitContract(t *testing.T) {
	off := func() (bool, error) { return false, nil }
	fnn, err := NewFnn(&stubSite{}, 1e3, 1e3, off, nil, nil, off, 0)
	require.NoError(t, err)

	require.Nil(t, fnn.MaxConsumptionPower())
	require.Nil(t, fnn.MaxProductionPower())

	require.NoError(t, fnn.runDim())
	require.NoError(t, fnn.runCurtail())

	require.NotNil(t, fnn.MaxConsumptionPower())
	require.Equal(t, 0.0, *fnn.MaxConsumptionPower())
	require.NotNil(t, fnn.MaxProductionPower())
	require.Equal(t, 0.0, *fnn.MaxProductionPower())
}

// TestFnnEdgeTriggered verifies a limit is only applied on a genuine
// transition, not every steady-state tick (a fresh pointer would prove a re-run).
func TestFnnEdgeTriggered(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	on := func() (bool, error) { return true, nil }
	fnn, err := NewFnn(&stubSite{}, 1e3, 1e3, nil, nil, nil, on, 0)
	require.NoError(t, err)

	require.NoError(t, fnn.runDim())
	first := fnn.consumptionLimit
	require.NotNil(t, first)

	require.NoError(t, fnn.runDim())
	require.Same(t, first, fnn.consumptionLimit, "setConsumptionLimit must not re-run on an unchanged tick")
}

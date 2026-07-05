package fnn

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

// TestFnnNoLimitContract verifies api.HEMS's "nil = no limit" contract holds
// for a connected-but-idle Fnn (no dim/curtail command received yet).
func TestFnnNoLimitContract(t *testing.T) {
	fnn, err := NewFnn(nil, 1e3, 1e3, nil, nil, nil, func() (bool, error) {
		return true, nil
	}, 0)
	require.NoError(t, err)

	require.Nil(t, fnn.MaxConsumptionPower())
	require.Nil(t, fnn.MaxProductionPower())
}

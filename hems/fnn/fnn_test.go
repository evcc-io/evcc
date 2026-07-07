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

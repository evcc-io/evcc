package charger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// without a mode getter the cached mode must follow the last set mode
func TestSgReadyDimCachesMode(t *testing.T) {
	var mode int64
	wb, err := NewSgReady(context.Background(), &embed{}, func(m int64) error {
		mode = m
		return nil
	}, nil, nil)
	require.NoError(t, err)

	require.NoError(t, wb.Dim(true))
	assert.Equal(t, Dim, mode)
	dimmed, err := wb.Dimmed()
	require.NoError(t, err)
	assert.True(t, dimmed)

	require.NoError(t, wb.Dim(false))
	assert.Equal(t, Normal, mode)
	dimmed, err = wb.Dimmed()
	require.NoError(t, err)
	assert.False(t, dimmed)
}

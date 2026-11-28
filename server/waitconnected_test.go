package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type waiter struct{ connected chan struct{} }

func (w waiter) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-w.connected:
		return nil
	}
}

func TestWaitConnected(t *testing.T) {
	assert.NoError(t, waitConnected(context.Background(), struct{}{}), "devices without Wait")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, waitConnected(ctx, waiter{make(chan struct{})}), context.DeadlineExceeded)

	connected := make(chan struct{})
	close(connected)
	assert.NoError(t, waitConnected(context.Background(), waiter{connected}))
}

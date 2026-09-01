package eebus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectorWaitUseCase(t *testing.T) {
	c := NewConnector()
	c.Connect(true)
	require.NoError(t, c.Wait(t.Context()))

	// use case data arrives after the connection
	go func() {
		time.Sleep(10 * time.Millisecond)
		c.UseCase()
	}()

	start := time.Now()
	c.WaitUseCase(t.Context())
	assert.Less(t, time.Since(start), useCaseTimeout, "should return on the use case signal, not the timeout")
}

func TestConnectorWaitUseCaseCancelled(t *testing.T) {
	c := NewConnector()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	start := time.Now()
	c.WaitUseCase(ctx)
	assert.Less(t, time.Since(start), useCaseTimeout, "should return on context cancellation")
}

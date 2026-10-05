package eebus

import (
	"context"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
)

const (
	registerTimeout = 90 * time.Second
	useCaseTimeout  = 5 * time.Second
)

type Connector struct {
	once     sync.Once
	connectC chan struct{}

	useCaseOnce sync.Once
	useCaseC    chan struct{}
}

func NewConnector() *Connector {
	return &Connector{
		connectC: make(chan struct{}),
		useCaseC: make(chan struct{}),
	}
}

func (c *Connector) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(registerTimeout):
		return api.ErrTimeout
	case <-c.connectC:
		return nil
	}
}

func (c *Connector) Connect(connected bool) {
	if connected {
		c.once.Do(func() { close(c.connectC) })
	}
}

// UseCase signals that the device's use case data has arrived.
func (c *Connector) UseCase() {
	c.useCaseOnce.Do(func() { close(c.useCaseC) })
}

// WaitUseCase waits for the device's use case data, which the remote announces
// only after the SHIP connection completes. A timeout is not an error, a remote
// may announce nothing usable.
func (c *Connector) WaitUseCase(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(useCaseTimeout):
	case <-c.useCaseC:
	}

	return nil
}

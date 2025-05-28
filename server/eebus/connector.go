package eebus

import (
	"context"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
)

const registerTimeout = 90 * time.Second

type Connector struct {
	once     sync.Once
	connectC chan struct{}
}

func NewConnector() *Connector {
	return &Connector{connectC: make(chan struct{})}
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

package eebus

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
)

type Connector struct {
	once     sync.Once
	connectC chan struct{}
}

func NewConnector() *Connector {
	return &Connector{connectC: make(chan struct{})}
}

func (c *Connector) Wait(timeout time.Duration) error {
	select {
	case <-time.After(timeout):
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

package eebus

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
)

type Connector struct {
	cb func(connected bool)

	mu        sync.RWMutex
	once      sync.Once
	connected bool
	connectC  chan struct{}
}

func NewConnector(cb func(connected bool)) *Connector {
	return &Connector{
		cb:       cb,
		connectC: make(chan struct{}),
	}
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
	c.mu.Lock()
	c.connected = connected
	c.mu.Unlock()

	if connected {
		c.once.Do(func() { close(c.connectC) })
	}

	if c.cb != nil {
		c.cb(connected)
	}
}

func (c *Connector) Connected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

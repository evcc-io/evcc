package core

import (
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

const wakeupTimeout = 30 * time.Second

// Timer measures active time between start and stop events
type Timer struct {
	sync.Mutex
	clck    clock.Clock
	started time.Time
}

// NewTimer creates timer that can expire
func NewTimer() *Timer {
	return &Timer{
		clck: clock.New(),
	}
}

// Start starts the timer if not started already
func (m *Timer) Start() {
	m.Lock()
	defer m.Unlock()

	if !m.started.IsZero() {
		return
	}

	m.started = m.clck.Now()
}

// Reset resets the timer
func (m *Timer) Stop() {
	m.Lock()
	defer m.Unlock()

	if m.started.IsZero() {
		return
	}

	m.started = time.Time{}
}

// Expired checks if the timer has elapsed and if resets its status
func (m *Timer) Expired() bool {
	m.Lock()
	defer m.Unlock()

	if m.started.IsZero() {
		return false
	}

	res := m.clck.Since(m.started) >= wakeupTimeout
	if res {
		m.started = time.Time{}
	}

	return res
}

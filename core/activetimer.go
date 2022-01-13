package core

import (
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

// ActiveTimer measures active time between start and stop events
type ActiveTimer struct {
	sync.Mutex
	clck clock.Clock

	active       bool
	started      time.Time
	lastduration int64
	called       bool
}

// NewActiveTimer creates ActiveTimer
// for tracking duration of an active Loadpoint
func NewActiveTimer() *ActiveTimer {
	return &ActiveTimer{
		clck: clock.New(),
	}
}

// start the active timer if not started already
func (m *ActiveTimer) Start() {
	m.Lock()
	defer m.Unlock()
	if !m.active {
		m.active = true
		m.started = m.clck.Now()
		m.called = false
	}
}

// stop and reset called
func (m *ActiveTimer) Reset() {
	m.Stop()
	m.called = false
}

// stop the active timer and save the duration
func (m *ActiveTimer) Stop() {
	m.Lock()
	defer m.Unlock()
	if m.active {
		m.active = false
		m.lastduration = int64(m.clck.Since(m.started).Seconds())
		m.called = true
	}
}

// returns the duration of the last started timer
func (m *ActiveTimer) duration() int64 {
	m.Lock()
	defer m.Unlock()

	if m.active {
		return int64(m.clck.Since(m.started).Seconds())
	}
	return m.lastduration
}

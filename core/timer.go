package core

import (
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

const wakeupTimeout = 30 * time.Second

const wakeupAttempts = 6 // wakeupAttempts is the count of wakeup attempts

// Timer measures active time between start and stop events
type Timer struct {
	sync.Mutex
	clck               clock.Clock
	started            time.Time
	wakeupAttemptsLeft int
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

	m.wakeupAttemptsLeft = wakeupAttempts

	if !m.started.IsZero() {
		return
	}

	m.started = m.clck.Now()
}

// Reset resets the timer
func (m *Timer) Stop() {
	m.Lock()
	defer m.Unlock()

	m.started = time.Time{}
}

// Expired checks if the timer has elapsed and if resets its status
func (m *Timer) Expired() bool {
	m.Lock()
	defer m.Unlock()

	res := !m.started.IsZero() && (m.clck.Since(m.started) >= wakeupTimeout)
	if res {
		m.wakeupAttemptsLeft--
		if m.wakeupAttemptsLeft == 0 {
			m.started = time.Time{}
		} else {
			m.started = m.clck.Now()
		}
	}

	return res
}

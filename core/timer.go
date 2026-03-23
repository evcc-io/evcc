package core

import (
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

const (
	wakeupTimeout  = 30 * time.Second
	wakeupAttempts = 6
)

type WakeUpEvent int

const (
	WakeUpTimerInactive WakeUpEvent = iota
	WakeUpTimerElapsed
	WakeUpTimerFinished
)

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

func (m *Timer) Running() bool {
	m.Lock()
	defer m.Unlock()

	return !m.started.IsZero()
}

// Elapsed checks if the timer has elapsed and if resets its status
func (m *Timer) Elapsed() WakeUpEvent {
	m.Lock()
	defer m.Unlock()

	if m.started.IsZero() || m.clck.Since(m.started) < wakeupTimeout {
		return WakeUpTimerInactive
	}

	if m.wakeupAttemptsLeft == 0 {
		m.started = time.Time{}
		return WakeUpTimerFinished
	}

	m.wakeupAttemptsLeft--

	m.started = m.clck.Now()
	return WakeUpTimerElapsed
}

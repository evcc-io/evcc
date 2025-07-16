package wrapper

import (
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

// ChargeTimer measures charging time between start and stop events
type ChargeTimer struct {
	mu    sync.Mutex
	clock clock.Clock

	charging bool
	start    time.Time
	duration time.Duration
}

// NewChargeTimer creates ChargeTimer for tracking duration between
// start and stop events
func NewChargeTimer() *ChargeTimer {
	return &ChargeTimer{
		clock: clock.New(),
	}
}

// StartCharge signals charge timer start
func (m *ChargeTimer) StartCharge(continued bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.start = m.clock.Now()

	if continued {
		m.charging = true
	} else {
		m.duration = 0
	}
}

// StopCharge signals charge timer stop
func (m *ChargeTimer) StopCharge() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.charging = false
	m.duration += m.clock.Since(m.start)
}

var _ ChargeResetter = (*ChargeTimer)(nil)

// ChargeResetter resets the charging session
func (m *ChargeTimer) ResetCharge() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.duration = 0
}

// ChargeDuration implements the api.ChargeTimer interface
func (m *ChargeTimer) ChargeDuration() (time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.charging {
		return m.duration + m.clock.Since(m.start), nil
	}
	return m.duration, nil
}

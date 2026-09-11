package wrapper

import (
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
)

// ChargeTimer measures charging time between start and stop events
type ChargeTimer struct {
	sync.Mutex
	clck     clock.Clock
	upstream api.ChargeTimer // charger-provided duration, takes precedence

	charging bool
	start    time.Time
	duration time.Duration
}

// NewChargeTimer creates ChargeTimer for tracking duration between
// start and stop events. Duration is taken from upstream if given.
func NewChargeTimer(upstream api.ChargeTimer) *ChargeTimer {
	return &ChargeTimer{
		clck:     clock.New(),
		upstream: upstream,
	}
}

// StartCharge signals charge timer start
func (m *ChargeTimer) StartCharge(continued bool) {
	m.Lock()
	defer m.Unlock()

	if m.upstream != nil {
		return
	}

	m.start = m.clck.Now()

	if continued {
		m.charging = true
	} else {
		m.duration = 0
	}
}

// StopCharge signals charge timer stop
func (m *ChargeTimer) StopCharge() {
	m.Lock()
	defer m.Unlock()

	if m.upstream != nil {
		return
	}

	m.charging = false
	m.duration += m.clck.Since(m.start)
}

// ResetCharge resets the charging session
func (m *ChargeTimer) ResetCharge() {
	m.Lock()
	defer m.Unlock()

	if m.upstream != nil {
		return
	}

	m.duration = 0
}

// ChargeDuration implements the api.ChargeTimer interface
func (m *ChargeTimer) ChargeDuration() (time.Duration, error) {
	m.Lock()
	defer m.Unlock()

	if m.upstream != nil {
		return m.upstream.ChargeDuration()
	}

	if m.charging {
		return m.duration + m.clck.Since(m.start), nil
	}
	return m.duration, nil
}

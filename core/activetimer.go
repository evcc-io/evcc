package core

import (
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const wakeUpWaitTime = 30 * time.Second

// ActiveTimer measures active time between start and stop events
type ActiveTimer struct {
	sync.Mutex
	clck         clock.Clock
	log          *util.Logger
	started      time.Time
	lastduration time.Duration
}

// NewActiveTimer creates ActiveTimer
// for tracking duration of an active Loadpoint
func NewActiveTimer(log *util.Logger) *ActiveTimer {
	return &ActiveTimer{
		clck: clock.New(),
		log:  log,
	}
}

// Start starts the active timer if not started already
func (m *ActiveTimer) Start() {
	m.Lock()
	defer m.Unlock()

	if !m.started.IsZero() || m.lastduration > 0 {
		return
	}

	m.started = m.clck.Now()
	m.log.DEBUG.Printf("wakeUp: start")
}

// stop and reset called
func (m *ActiveTimer) Reset() {
	m.Lock()
	defer m.Unlock()

	if m.started.IsZero() && m.lastduration == 0 {
		return
	}

	m.lastduration = 0
	m.started = time.Time{}
	m.log.DEBUG.Printf("wakeUp: reset")
}

// stop the active timer and save the duration
func (m *ActiveTimer) Stop() {
	m.Lock()
	defer m.Unlock()

	if m.started.IsZero() {
		return
	}

	m.lastduration = m.clck.Since(m.started)
	m.started = time.Time{}
	m.log.DEBUG.Printf("wakeUp: stop")
}

// wakeUp logic
func (m *ActiveTimer) WakeUp(charger api.Charger, vehicle api.Vehicle) {
	if m.started.IsZero() {
		return
	}

	if m.clck.Since(m.started) > wakeUpWaitTime {
		m.log.DEBUG.Printf("wakeUp: active since %.0fs", m.clck.Since(m.started).Seconds())

		// charger
		if c, ok := charger.(api.AlarmClock); ok {
			if err := c.WakeUp(); err != nil {
				m.log.ERROR.Printf("wakeUp charger: %v", err)
			}
		}

		// vehicle
		if vs, ok := vehicle.(api.AlarmClock); ok {
			if err := vs.WakeUp(); err != nil {
				m.log.ERROR.Printf("wakeUp vehicle: %v", err)
			}
		}

		m.Stop()
	}
}

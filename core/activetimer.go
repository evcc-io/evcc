package core

import (
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	wakeUpWaitTime = 30
)

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

// start the active timer if not started already
func (m *ActiveTimer) Start() {
	if !m.started.IsZero() {
		return
	}
	if m.lastduration > 0 {
		return
	}
	m.Lock()
	defer m.Unlock()
	m.started = m.clck.Now()
	m.log.DEBUG.Printf("WakeUp Timer started")
}

// stop and reset called
func (m *ActiveTimer) Reset() {
	if m.started.IsZero() && m.lastduration == 0 {
		return
	}
	m.Lock()
	defer m.Unlock()
	m.lastduration = 0
	m.started = time.Time{}
	m.log.DEBUG.Printf("WakeUp Timer resetted")
}

// stop the active timer and save the duration
func (m *ActiveTimer) Stop() {
	if m.started.IsZero() {
		return
	}
	m.Lock()
	defer m.Unlock()
	m.lastduration = m.clck.Since(m.started)
	m.started = time.Time{}
	m.log.DEBUG.Printf("WakeUp Timer stopped")
}

// wakeUp logic
func (m *ActiveTimer) WakeUp(charger api.Charger, vehicle api.Vehicle) {
	if m.started.IsZero() {
		return
	}
	if m.clck.Since(m.started).Seconds() > wakeUpWaitTime {
		m.log.DEBUG.Printf("time for WakeUp calls - sleeping? WakeUpTimer active:%ds", int(m.clck.Since(m.started).Seconds()))
		// call the Charger WakeUp if available
		if c, ok := charger.(api.AlarmClock); ok {
			if err := c.WakeUp(); err == nil {
				m.log.DEBUG.Printf("charger WakeUp called")
			} else {
				m.log.ERROR.Printf("charger wakeup error :  %v", err)
			}
		}
		// call the Vehicle WakeUp if available
		if vs, ok := vehicle.(api.AlarmClock); ok {
			if err := vs.WakeUp(); err == nil {
				m.log.DEBUG.Printf("vehicle WakeUp API called")
			} else {
				m.log.ERROR.Printf("vehicle wakeup error :  %v", err)
			}
		}
		m.Stop()
	}
}

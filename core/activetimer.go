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
	clck clock.Clock

	started      time.Time
	lastduration time.Duration
}

// NewActiveTimer creates ActiveTimer
// for tracking duration of an active Loadpoint
func NewActiveTimer() *ActiveTimer {
	return &ActiveTimer{
		clck: clock.New(),
	}
}

// start the active timer if not started already
func (m *ActiveTimer) Start(log *util.Logger) {
	if !m.started.IsZero() {
		return
	}
	if m.lastduration > 0 {
		return
	}
	m.Lock()
	defer m.Unlock()
	log.DEBUG.Printf("start WakeUp Timer")
	m.started = m.clck.Now()
}

// stop and reset called
func (m *ActiveTimer) Reset() {
	m.Lock()
	defer m.Unlock()
	m.lastduration = 0
	m.started = time.Time{}
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
}

// wakeUp logic
func (m *ActiveTimer) WakeUp(charger api.Charger, vehicle api.Vehicle, log *util.Logger) {
	if m.started.IsZero() {
		return
	}
	if m.clck.Since(m.started).Seconds() > wakeUpWaitTime {
		log.DEBUG.Printf("time for WakeUp calls - sleeping? WakeUpTimer active:%ds", int(m.clck.Since(m.started).Seconds()))
		// call the Charger WakeUp if available
		if c, ok := charger.(api.AlarmClock); ok {
			if err := c.WakeUp(); err == nil {
				log.DEBUG.Printf("charger WakeUp called")
			}
		}
		// call the Vehicle WakeUp if available
		if vs, ok := vehicle.(api.AlarmClock); ok {
			if err := vs.WakeUp(); err == nil {
				log.DEBUG.Printf("vehicle WakeUp API called")
			}
		}
		m.Stop()
	}
}

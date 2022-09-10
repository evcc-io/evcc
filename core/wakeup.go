package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
)

const (
	wakeupTimeout = 30 * time.Second

	stageCharger = iota
	stageVehicle
)

// WakeUp measures active time between start and stop events
type WakeUp struct {
	sync.Mutex
	pub     publisher
	clck    clock.Clock
	started time.Time
	stage   int // 0 is charger, 1 is vehicle
	charger api.Resurrector
	vehicle api.Resurrector
}

// NewWakeUp creates timer that can expire
func NewWakeUp(pub publisher, charger api.Charger) *WakeUp {
	m := &WakeUp{
		clck: clock.New(),
		pub:  pub,
	}

	if r, ok := charger.(api.Resurrector); ok {
		m.charger = r
	}

	return m
}

// Start starts the timer if not started already
func (m *WakeUp) Start(vehicle api.Vehicle) {
	// test guard
	if m == nil {
		return
	}

	m.Lock()
	defer m.Unlock()

	m.vehicle = nil
	if r, ok := vehicle.(api.Resurrector); ok {
		m.vehicle = r
	}

	if m.started.IsZero() && (m.charger != nil || m.vehicle != nil) {
		m.started = m.clck.Now()
		m.pub.publish("wakeupTimer", wakeupTimeout)

		// start with vehicle if charger is not supported
		m.stage = stageCharger
		if m.charger == nil {
			m.stage = stageVehicle
		}
	}
}

// Reset resets the timer
func (m *WakeUp) Stop() {
	// test guard
	if m == nil {
		return
	}

	m.Lock()
	defer m.Unlock()

	m.started = time.Time{}
	m.pub.publish("wakeupTimer", 0)
}

// wakeUpVehicle calls the next available wake-up method
func (m *WakeUp) WakeUp() error {
	// test guard
	if m == nil {
		return nil
	}

	m.Lock()
	defer m.Unlock()

	if m.started.IsZero() {
		return nil
	}

	if since := m.clck.Since(m.started); since < wakeupTimeout {
		m.pub.publish("wakeupTimer", wakeupTimeout-since)
		return nil
	}

	var err error

	switch m.stage {
	case stageCharger:
		if err = m.charger.WakeUp(); err != nil {
			err = fmt.Errorf("wake up charger: %w", err)
		}

		if m.vehicle != nil {
			m.stage = stageVehicle
			m.started = m.clck.Now()
			m.pub.publish("wakeupTimer", wakeupTimeout)
		} else {
			m.started = time.Time{}
			m.pub.publish("wakeupTimer", 0)
		}

	case stageVehicle:
		if err = m.vehicle.WakeUp(); err != nil {
			err = fmt.Errorf("wake up vehicle: %w", err)
		}

		m.started = time.Time{}
		m.pub.publish("wakeupTimer", 0)
	}

	return err
}

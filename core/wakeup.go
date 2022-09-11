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

// WakeUp is a staged charger/vehicle timer-based wakeup mechanism
type WakeUp struct {
	sync.Mutex
	pub     publisher
	clck    clock.Clock
	started time.Time
	stage   int // 0 is charger, 1 is vehicle
	charger api.Resurrector
	vehicle api.Resurrector
}

// NewWakeUp creates a staged charger/vehicle timer-based wakeup mechanism
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

// Start starts the timer for given vehicles if not already started
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

// Stop stops the timer if not already stopped
func (m *WakeUp) Stop() {
	// test guard
	if m == nil {
		return
	}

	m.Lock()
	defer m.Unlock()

	if !m.started.IsZero() {
		m.started = time.Time{}
		m.pub.publish("wakeupTimer", 0)
	}
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

	var (
		err         error
		wakeupTimer time.Duration
	)

	// expire timer
	m.started = time.Time{}

	switch m.stage {
	case stageCharger:
		if err = m.charger.WakeUp(); err != nil {
			err = fmt.Errorf("wake up charger: %w", err)
		}

		// start vehicle stage
		if m.vehicle != nil {
			m.stage = stageVehicle
			m.started = m.clck.Now()
			wakeupTimer = wakeupTimeout
		}

	case stageVehicle:
		if err = m.vehicle.WakeUp(); err != nil {
			err = fmt.Errorf("wake up vehicle: %w", err)
		}
	}

	m.pub.publish("wakeupTimer", wakeupTimer)

	return err
}

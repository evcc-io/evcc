package ocpp

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/evcc-io/evcc/util"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
)

type registration struct {
	mu        sync.RWMutex
	setup     sync.RWMutex                            // serialises chargepoint setup
	cp        *CP                                     // guarded by setup and CS mutexes
	connected bool                                    // WebSocket transport connected; guarded by CS mutex
	status    map[int]*core.StatusNotificationRequest // guarded by mu mutex
}

func newRegistration() *registration {
	return &registration{status: make(map[int]*core.StatusNotificationRequest)}
}

type CS struct {
	ocpp16.CentralSystem
	mu          sync.Mutex
	log         *util.Logger
	regs        map[string]*registration // guarded by mu mutex
	txnId       atomic.Int64
	publishFunc func()
}

type stationStatus struct {
	ID     string        `json:"id"`
	Status StationStatus `json:"status"`
}

// Status represents the runtime OCPP status
type Status struct {
	ExternalUrl string          `json:"externalUrl,omitempty"`
	Stations    []stationStatus `json:"stations"`
}

// status returns the OCPP runtime status
func (cs *CS) status() Status {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	stations := []stationStatus{}

	for id, reg := range cs.regs {
		if id == "" {
			continue // skip anonymous registrations
		}

		state := StationStatusUnknown
		if cp := reg.cp; cp != nil {
			if cp.Connected() {
				state = StationStatusConnected
			} else {
				state = StationStatusConfigured
			}
		}

		stations = append(stations, stationStatus{
			ID:     id,
			Status: state,
		})
	}

	return Status{
		ExternalUrl: ExternalUrl(),
		Stations:    stations,
	}
}

// SetUpdated sets a callback function that is called when the status changes
func (cs *CS) SetUpdated(f func()) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.publishFunc = f
}

// errorHandler logs error channel
func (cs *CS) errorHandler(errC <-chan error) {
	for err := range errC {
		cs.log.ERROR.Println(err)
	}
}

// publish triggers the publish callback if set
func (cs *CS) publish() {
	if cs.publishFunc != nil {
		cs.publishFunc()
	}
}

func (cs *CS) ChargepointByID(id string) (*CP, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	reg, ok := cs.regs[id]
	if !ok {
		return nil, fmt.Errorf("unknown charge point: %s", id)
	}
	if reg.cp == nil {
		return nil, fmt.Errorf("charge point not configured: %s", id)
	}
	return reg.cp, nil
}

func (cs *CS) WithConnectorStatus(id string, connector int, fun func(status *core.StatusNotificationRequest)) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if reg, ok := cs.regs[id]; ok {
		reg.mu.RLock()
		if status, ok := reg.status[connector]; ok {
			fun(status)
		}
		reg.mu.RUnlock()
	}
}

// RegisterChargepoint registers a charge point with the central system of returns an already registered charge point
func (cs *CS) RegisterChargepoint(id string, newfun func() *CP, init func(*CP) error) (*CP, error) {
	cs.mu.Lock()

	// prepare shadow state
	reg, ok := cs.regs[id]
	if !ok {
		reg = newRegistration()
		cs.regs[id] = reg
	}

	cs.mu.Unlock()
	cs.publish()

	// serialise on chargepoint id
	reg.setup.Lock()
	defer reg.setup.Unlock()

	cs.mu.Lock()
	cp := reg.cp
	cs.mu.Unlock()

	// setup already completed?
	if cp != nil {
		// duplicate registration of id empty
		if id == "" {
			return nil, errors.New("cannot have >1 charge point with empty station id")
		}

		return cp, nil
	}

	// first time - create the charge point
	cp = newfun()

	// Publish cp and check transport state atomically. NewChargePoint may have
	// run before we created the reg (reg.connected=false here, NewChargePoint
	// will see reg.cp and call onTransportConnect) or after we publish reg.cp
	// (reg.connected=true here, we call onTransportConnect ourselves). The
	// shared cs.mu serialises both code paths so exactly one of them fires.
	cs.mu.Lock()
	reg.cp = cp
	wasConnected := reg.connected
	cs.mu.Unlock()

	if wasConnected {
		cp.onTransportConnect()
	}

	return cp, init(cp)
}

// NewChargePoint implements ocpp16.ChargePointConnectionHandler
func (cs *CS) NewChargePoint(chargePoint ocpp16.ChargePointConnection) {
	cs.mu.Lock()

	// check for configured charge point
	reg, ok := cs.regs[chargePoint.ID()]
	if ok {
		cs.log.DEBUG.Printf("charge point connected: %s", chargePoint.ID())

		// record transport state so RegisterChargepoint sees it once cp is set
		reg.connected = true

		// wait for BootNotification before marking as connected
		if cp := reg.cp; cp != nil {
			cp.onTransportConnect()
		}

		cs.mu.Unlock()
		cs.publish()
		return
	}

	// check for configured anonymous charge point
	reg, ok = cs.regs[""]
	if ok && reg.cp != nil {
		cp := reg.cp
		cs.log.INFO.Printf("charge point connected, registering: %s", chargePoint.ID())

		// update id
		cp.RegisterID(chargePoint.ID())
		reg.connected = true
		cs.regs[chargePoint.ID()] = reg
		delete(cs.regs, "")

		cp.onTransportConnect()

		cs.mu.Unlock()
		cs.publish()
		return
	}

	// register unknown charge point
	reg = newRegistration()
	reg.connected = true
	cs.regs[chargePoint.ID()] = reg
	cs.log.INFO.Printf("unknown charge point connected: %s", chargePoint.ID())

	cs.mu.Unlock()
	cs.publish()
}

// ChargePointDisconnected implements ocpp16.ChargePointConnectionHandler
func (cs *CS) ChargePointDisconnected(chargePoint ocpp16.ChargePointConnection) {
	cs.log.DEBUG.Printf("charge point disconnected: %s", chargePoint.ID())

	cs.mu.Lock()
	if reg, ok := cs.regs[chargePoint.ID()]; ok {
		reg.connected = false
	}
	cs.mu.Unlock()

	if cp, err := cs.ChargepointByID(chargePoint.ID()); err == nil {
		cp.connect(false)
	}

	cs.publish()
}

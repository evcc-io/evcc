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
	mu     sync.RWMutex
	setup  sync.RWMutex                            // serialises chargepoint setup
	cp     *CP                                     // guarded by setup and CS mutexes
	status map[int]*core.StatusNotificationRequest // guarded by mu mutex
}

func newRegistration() *registration {
	return &registration{status: make(map[int]*core.StatusNotificationRequest)}
}

type CS struct {
	ocpp16.CentralSystem
	mu    sync.Mutex
	log   *util.Logger
	regs  map[string]*registration // guarded by mu mutex
	txnId atomic.Int64
}

// errorHandler logs error channel
func (cs *CS) errorHandler(errC <-chan error) {
	for err := range errC {
		cs.log.ERROR.Println(err)
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
	reg, registered := cs.regs[id]
	if !registered {
		reg = newRegistration()
		cs.regs[id] = reg
	}

	cs.mu.Unlock()

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

	// first time- create the charge point
	cp = newfun()

	cs.mu.Lock()
	reg.cp = cp
	cs.mu.Unlock()

	if registered {
		cp.connect(true)
	}

	return cp, init(cp)
}

// NewChargePoint implements ocpp16.ChargePointConnectionHandler
func (cs *CS) NewChargePoint(chargePoint ocpp16.ChargePointConnection) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// check for configured charge point
	reg, ok := cs.regs[chargePoint.ID()]
	if ok {
		cs.log.DEBUG.Printf("charge point connected: %s", chargePoint.ID())

		// trigger initial connection if charge point is already setup
		if cp := reg.cp; cp != nil {
			cp.connect(true)
		}

		return
	}

	// check for configured anonymous charge point
	reg, ok = cs.regs[""]
	if ok && reg.cp != nil {
		cp := reg.cp
		cs.log.INFO.Printf("charge point connected, registering: %s", chargePoint.ID())

		// update id
		cp.RegisterID(chargePoint.ID())
		cs.regs[chargePoint.ID()] = reg
		delete(cs.regs, "")

		cp.connect(true)

		return
	}

	cs.log.WARN.Printf("unknown charge point connected: %s", chargePoint.ID())

	// register unknown charge point
	// when charge point setup is complete, it will eventually be associated with the connected id
	cs.regs[chargePoint.ID()] = newRegistration()
}

// ChargePointDisconnected implements ocpp16.ChargePointConnectionHandler
func (cs *CS) ChargePointDisconnected(chargePoint ocpp16.ChargePointConnection) {
	cs.log.DEBUG.Printf("charge point disconnected: %s", chargePoint.ID())

	if cp, err := cs.ChargepointByID(chargePoint.ID()); err == nil {
		cp.connect(false)
	}
}

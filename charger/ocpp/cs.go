package ocpp

import (
	"errors"
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/util"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
)

type CS struct {
	mu  sync.Mutex
	log *util.Logger
	ocpp16.CentralSystem
	cps map[string]*CP
}

// Register registers a charge point with the central system.
// The charge point identified by id may already be connected in which case initial connection is triggered.
func (cs *CS) Register(id string, cp *CP) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, ok := cs.cps[id]; ok && id == "" {
		return errors.New("cannot have >1 charge point with empty station id")
	}

	// trigger unknown charge point connected
	if unknown, ok := cs.cps[id]; ok && unknown == nil {
		cp.connect(true)
	}

	cs.cps[id] = cp

	return nil
}

// errorHandler logs error channel
func (cs *CS) errorHandler(errC <-chan error) {
	for err := range errC {
		cs.log.ERROR.Println(err)
	}
}

// chargepointByID returns a configured charge point identified by id.
func (cs *CS) chargepointByID(id string) (*CP, error) {
	cp, ok := cs.cps[id]
	if !ok {
		return nil, fmt.Errorf("unknown charge point: %s", id)
	}
	if cp == nil {
		return nil, fmt.Errorf("charge point not configured: %s", id)
	}
	return cp, nil
}

// NewChargePoint implements ocpp16.ChargePointConnectionHandler
func (cs *CS) NewChargePoint(chargePoint ocpp16.ChargePointConnection) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// check for configured charge point
	cp, ok := cs.cps[chargePoint.ID()]
	if ok {
		cs.log.DEBUG.Printf("charge point connected: %s", chargePoint.ID())

		// trigger initial connection if charge point is already setup
		if cp != nil {
			cp.connect(true)
		}

		return
	}

	// check for configured anonymous charge point
	cp, ok = cs.cps[""]
	if ok && cp != nil {
		cs.log.INFO.Printf("charge point connected, registering: %s", chargePoint.ID())

		// update id
		cp.RegisterID(chargePoint.ID())

		cs.cps[chargePoint.ID()] = cp
		delete(cs.cps, "")

		cp.connect(true)

		return
	}

	cs.log.WARN.Printf("unknown charge point connected: %s", chargePoint.ID())

	// register unknown charge point
	// when charge point setup is complete, it will eventually be associated with the connected id
	cs.cps[chargePoint.ID()] = nil
}

// ChargePointDisconnected implements ocpp16.ChargePointConnectionHandler
func (cs *CS) ChargePointDisconnected(chargePoint ocpp16.ChargePointConnection) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.log.DEBUG.Printf("charge point disconnected: %s", chargePoint.ID())

	if cp, err := cs.chargepointByID(chargePoint.ID()); err != nil {
		cp.connect(false)
	}
}

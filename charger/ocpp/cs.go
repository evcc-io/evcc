package ocpp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/util"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"golang.org/x/sync/semaphore"
)

type CS struct {
	mu  sync.Mutex
	log *util.Logger
	ocpp16.CentralSystem
	cps   map[string]*CP
	init  map[string]*semaphore.Weighted
	txnId int
}

// Register registers a charge point with the central system.
// The charge point identified by id may already be connected in which case initial connection is triggered.
func (cs *CS) register(id string, new *CP) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cp, ok := cs.cps[id]

	// case 1: charge point neither registered nor physically connected
	if !ok {
		cs.cps[id] = new
		return nil
	}

	// case 2: duplicate registration of id empty
	if id == "" {
		return errors.New("cannot have >1 charge point with empty station id")
	}

	// case 3: charge point not registered but physically already connected
	if cp == nil {
		cs.cps[id] = new
		new.connect(true)
	}

	return nil
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

	cp, ok := cs.cps[id]
	if !ok {
		return nil, fmt.Errorf("unknown charge point: %s", id)
	}
	if cp == nil {
		return nil, fmt.Errorf("charge point not configured: %s", id)
	}
	return cp, nil
}

func (cs *CS) RegisterChargepoint(id string, new func() *CP, init func(*CP) error) (*CP, error) {
	cs.mu.Lock()
	sem, ok := cs.init[id]
	if !ok {
		sem = semaphore.NewWeighted(1)
		cs.init[id] = sem
	}
	cs.mu.Unlock()

	// serialise on chargepoint id
	sem.Acquire(context.TODO(), 1)
	defer sem.Release(1)

	cp, err := cs.ChargepointByID(id)
	if err != nil {
		cp = new()
	}

	// should not error
	if err := cs.register(id, cp); err != nil {
		return nil, err
	}

	return cp, init(cp)
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
	cs.log.DEBUG.Printf("charge point disconnected: %s", chargePoint.ID())

	if cp, err := cs.ChargepointByID(chargePoint.ID()); err == nil {
		cp.connect(false)
	}
}

// NewTransactionID returns a CS-wide unique transactionId
func (cs *CS) NewTransactionID() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.txnId++
	return cs.txnId
}

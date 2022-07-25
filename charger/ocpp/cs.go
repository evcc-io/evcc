package ocpp

import (
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/util"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

type CS struct {
	mu  sync.Mutex
	log *util.Logger
	cs  ocpp16.CentralSystem
	cps map[string]*CP
}

func (cs *CS) Register(id string, meterSupported bool) *CP {
	cp := &CP{
		id:             id,
		log:            util.NewLogger("ocpp-cp"),
		measurements:   make(map[string]types.SampledValue),
		meterSupported: meterSupported,
	}

	cp.initialized = sync.NewCond(&cp.mu)

	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.cps[id] = cp

	return cp
}

// errorHandler logs error channel
func (cs *CS) errorHandler(errC <-chan error) {
	for err := range errC {
		cs.log.ERROR.Println(err)
	}
}

func (cs *CS) chargepointByID(id string) (*CP, error) {
	cp, ok := cs.cps[id]
	if !ok {
		return nil, fmt.Errorf("unknown charge point: %s", id)
	}
	return cp, nil
}

func (cs *CS) NewChargePoint(chargePoint ocpp16.ChargePointConnection) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, err := cs.chargepointByID(chargePoint.ID()); err != nil {
		cs.log.WARN.Println(err)

		cs.log.INFO.Printf("new chargepoint with ID (%s) detected, attempting to remap", chargePoint.ID())
		if unknownIDCP, ok := cs.cps[""]; ok && unknownIDCP != nil {
			cs.mu.Lock()
			unknownIDCP.id = chargePoint.ID()
			cs.cps[chargePoint.ID()] = unknownIDCP
			delete(cs.cps, "") // remove unknownID key
			cs.mu.Unlock()
		}
	}
}

func (cs *CS) ChargePointDisconnected(chargePoint ocpp16.ChargePointConnection) {
	if _, err := cs.chargepointByID(chargePoint.ID()); err != nil {
		cs.log.ERROR.Println(err)
	}
}

func (cs *CS) CS() ocpp16.CentralSystem {
	return cs.cs
}

func (cs *CS) Debug(args ...interface{}) {
	cs.log.TRACE.Println(args...)
}

func (cs *CS) Debugf(fmt string, args ...interface{}) {
	cs.log.TRACE.Printf(fmt, args...)
}

func (cs *CS) Info(args ...interface{}) {
	cs.log.DEBUG.Println(args...)
}

func (cs *CS) Infof(fmt string, args ...interface{}) {
	cs.log.DEBUG.Printf(fmt, args...)
}

func (cs *CS) Error(args ...interface{}) {
	cs.log.ERROR.Println(args...)
}

func (cs *CS) Errorf(fmt string, args ...interface{}) {
	cs.log.ERROR.Printf(fmt, args...)
}

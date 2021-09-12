package ocpp

import (
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/util"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
)

type CS struct {
	mu  sync.Mutex
	log *util.Logger
	cs  ocpp16.CentralSystem
	cps map[string]*CP
}

func (cs *CS) Register(id string) *CP {
	cp := &CP{
		id: id,
	}

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

	if cp, err := cs.chargepointByID(chargePoint.ID()); err != nil {
		cs.log.ERROR.Println(err)
	} else {
		cp.SetAvailable(true)
	}
}

func (cs *CS) ChargePointDisconnected(chargePoint ocpp16.ChargePointConnection) {
	if cp, err := cs.chargepointByID(chargePoint.ID()); err != nil {
		cs.log.ERROR.Println(err)
	} else {
		cp.SetAvailable(false)
	}
}

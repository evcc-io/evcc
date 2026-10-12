package ocpp2

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/evcc-io/evcc/util"
	ocpp2 "github.com/lorenzodonini/ocpp-go/ocpp2.0.1"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ws"
)

type registration2 struct {
	mu     sync.RWMutex
	setup  sync.RWMutex
	st     *Station
	status map[int]*availability.StatusNotificationRequest
}

func newRegistration2() *registration2 {
	return &registration2{
		status: make(map[int]*availability.StatusNotificationRequest),
	}
}

type CS2 struct {
	ocpp2.CSMS                        // Eingebettetes CSMS-Interface
	mu          sync.Mutex
	log         *util.Logger
	regs        map[string]*registration2
	txnId       atomic.Int64
	publishFunc func()
	server      ws.Server              // für Addr()-Check in listen()
}

func (cs *CS2) errorHandler(errC <-chan error) {
	for err := range errC {
		cs.log.ERROR.Println(err)
	}
}

func (cs *CS2) publish() {
	if cs.publishFunc != nil {
		cs.publishFunc()
	}
}

func (cs *CS2) SetUpdated(f func()) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.publishFunc = f
}

func (cs *CS2) StationByID(id string) (*Station, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	reg, ok := cs.regs[id]
	if !ok {
		return nil, fmt.Errorf("unknown charging station: %s", id)
	}
	if reg.st == nil {
		return nil, fmt.Errorf("charging station not yet configured: %s", id)
	}
	return reg.st, nil
}

func (cs *CS2) WithEVSEStatus(stationID string, evseId int, fun func(*availability.StatusNotificationRequest)) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if reg, ok := cs.regs[stationID]; ok {
		reg.mu.RLock()
		if s, ok := reg.status[evseId]; ok {
			fun(s)
		}
		reg.mu.RUnlock()
	}
}

func (cs *CS2) RegisterStation(id string, newfun func() *Station, init func(*Station) error) (*Station, error) {
	cs.mu.Lock()
	reg, registered := cs.regs[id]
	if !registered {
		reg = newRegistration2()
		cs.regs[id] = reg
	}
	cs.mu.Unlock()
	cs.publish()

	reg.setup.Lock()
	defer reg.setup.Unlock()

	cs.mu.Lock()
	st := reg.st
	cs.mu.Unlock()

	if st != nil {
		if id == "" {
			return nil, errors.New("cannot have >1 charging station with empty station id")
		}
		return st, nil
	}

	st = newfun()
	cs.mu.Lock()
	reg.st = st
	cs.mu.Unlock()

	if registered {
		st.onTransportConnect()
	}

	err := init(st)
	if err != nil {
		cs.mu.Lock()
		if reg.st == st {
			reg.st = nil
		}
		cs.mu.Unlock()
	}
	return st, err
}

func (cs *CS2) NewChargingStation(station ocpp2.ChargingStationConnection) {
	cs.mu.Lock()

	if reg, ok := cs.regs[station.ID()]; ok {
		cs.log.DEBUG.Printf("charging station connected: %s", station.ID())
		if st := reg.st; st != nil {
			st.onTransportConnect()
		}
		cs.mu.Unlock()
		cs.publish()
		return
	}

	if reg, ok := cs.regs[""]; ok && reg.st != nil {
		st := reg.st
		cs.log.INFO.Printf("charging station connected, registering: %s", station.ID())
		st.RegisterID(station.ID())
		cs.regs[station.ID()] = reg
		delete(cs.regs, "")
		st.onTransportConnect()
		cs.mu.Unlock()
		cs.publish()
		return
	}

	cs.regs[station.ID()] = newRegistration2()
	cs.log.INFO.Printf("unknown charging station connected: %s", station.ID())
	cs.mu.Unlock()
	cs.publish()
}

func (cs *CS2) ChargingStationDisconnected(station ocpp2.ChargingStationConnection) {
	cs.log.DEBUG.Printf("charging station disconnected: %s", station.ID())
	if st, err := cs.StationByID(station.ID()); err == nil {
		st.connect(false)
	}
	cs.publish()
}
package ocpp20

import (
	"errors"
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/util"
	ocpp2 "github.com/lorenzodonini/ocpp-go/ocpp2.0.1"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
)

type registration struct {
	mu     sync.RWMutex
	setup  sync.RWMutex                                    // serialises charging station setup
	cs     *Station                                        // guarded by setup and CSMS mutexes
	status map[int]*availability.StatusNotificationRequest // guarded by mu mutex
}

func newRegistration() *registration {
	return &registration{status: make(map[int]*availability.StatusNotificationRequest)}
}

// CSMS is the OCPP 2.0.1 Charging Station Management System
type CSMS struct {
	ocpp2.CSMS
	mu          sync.Mutex
	log         *util.Logger
	regs        map[string]*registration // guarded by mu mutex
	publishFunc func()
}

// status returns the OCPP 2.0.1 runtime status
func (cs *CSMS) status() []ocpp.StationInfo {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	stations := []ocpp.StationInfo{}

	for id, reg := range cs.regs {
		if id == "" {
			continue // skip anonymous registrations
		}

		state := ocpp.StationStatusUnknown
		if cp := reg.cs; cp != nil {
			if cp.Connected() {
				state = ocpp.StationStatusConnected
			} else {
				state = ocpp.StationStatusConfigured
			}
		}

		stations = append(stations, ocpp.StationInfo{
			ID:     id,
			Status: state,
		})
	}

	return stations
}

// SetUpdated sets a callback function that is called when the status changes
func (cs *CSMS) SetUpdated(f func()) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.publishFunc = f
}

// errorHandler logs error channel
func (cs *CSMS) errorHandler(errC <-chan error) {
	for err := range errC {
		cs.log.ERROR.Println(err)
	}
}

// publish triggers the publish callback if set
func (cs *CSMS) publish() {
	if cs.publishFunc != nil {
		cs.publishFunc()
	}
}

func (cs *CSMS) ChargingStationByID(id string) (*Station, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	reg, ok := cs.regs[id]
	if !ok {
		return nil, fmt.Errorf("unknown charging station: %s", id)
	}
	if reg.cs == nil {
		return nil, fmt.Errorf("charging station not configured: %s", id)
	}
	return reg.cs, nil
}

func (cs *CSMS) WithEVSEStatus(id string, evseID int, fun func(status *availability.StatusNotificationRequest)) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if reg, ok := cs.regs[id]; ok {
		reg.mu.RLock()
		if status, ok := reg.status[evseID]; ok {
			fun(status)
		}
		reg.mu.RUnlock()
	}
}

// RegisterChargingStation registers a charging station with the CSMS or returns an already registered one
func (cs *CSMS) RegisterChargingStation(id string, newfun func() *Station, init func(*Station) error) (*Station, error) {
	cs.mu.Lock()

	// prepare shadow state
	reg, registered := cs.regs[id]
	if !registered {
		reg = newRegistration()
		cs.regs[id] = reg
	}

	cs.mu.Unlock()
	cs.publish()

	// serialise on charging station id
	reg.setup.Lock()
	defer reg.setup.Unlock()

	cs.mu.Lock()
	cp := reg.cs
	cs.mu.Unlock()

	// setup already completed?
	if cp != nil {
		// duplicate registration of id empty
		if id == "" {
			return nil, errors.New("cannot have >1 charging station with empty station id")
		}

		return cp, nil
	}

	// first time- create the charging station
	cp = newfun()

	cs.mu.Lock()
	reg.cs = cp
	cs.mu.Unlock()

	if registered {
		cp.onTransportConnect()
	}

	return cp, init(cp)
}

// NewChargingStation implements ocpp2.ChargingStationConnectionHandler
func (cs *CSMS) NewChargingStation(station ocpp2.ChargingStationConnection) {
	cs.mu.Lock()

	// check for configured charging station
	reg, ok := cs.regs[station.ID()]
	if ok {
		cs.log.DEBUG.Printf("charging station connected: %s", station.ID())

		// wait for BootNotification before marking as connected
		if cp := reg.cs; cp != nil {
			cp.onTransportConnect()
		}

		cs.mu.Unlock()
		cs.publish()
		return
	}

	// check for configured anonymous charging station
	reg, ok = cs.regs[""]
	if ok && reg.cs != nil {
		cp := reg.cs
		cs.log.INFO.Printf("charging station connected, registering: %s", station.ID())

		if err := cp.RegisterID(station.ID()); err != nil {
			cs.log.ERROR.Printf("register charging station: %v", err)
			cs.mu.Unlock()
			cs.publish()
			return
		}
		cs.regs[station.ID()] = reg
		delete(cs.regs, "")

		cp.onTransportConnect()

		cs.mu.Unlock()
		cs.publish()
		return
	}

	// register unknown charging station
	cs.regs[station.ID()] = newRegistration()
	cs.log.INFO.Printf("unknown charging station connected: %s", station.ID())

	cs.mu.Unlock()
	cs.publish()
}

// ChargingStationDisconnected implements ocpp2.ChargingStationConnectionHandler
func (cs *CSMS) ChargingStationDisconnected(station ocpp2.ChargingStationConnection) {
	cs.log.DEBUG.Printf("charging station disconnected: %s", station.ID())

	if cp, err := cs.ChargingStationByID(station.ID()); err == nil {
		cp.connect(false)
	}

	cs.publish()
}

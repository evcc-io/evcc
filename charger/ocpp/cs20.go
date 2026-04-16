package ocpp

import (
	"errors"
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/util"
	ocpp2 "github.com/lorenzodonini/ocpp-go/ocpp2.0.1"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
)

type registration20 struct {
	mu     sync.RWMutex
	setup  sync.RWMutex                                   // serialises charging station setup
	cs     *CS20CP                                        // guarded by setup and CSMS mutexes
	status map[int]*availability.StatusNotificationRequest // guarded by mu mutex
}

func newRegistration20() *registration20 {
	return &registration20{status: make(map[int]*availability.StatusNotificationRequest)}
}

// CSMS20 is the OCPP 2.0.1 Charging Station Management System
type CSMS20 struct {
	ocpp2.CSMS
	mu          sync.Mutex
	log         *util.Logger
	regs        map[string]*registration20 // guarded by mu mutex
	publishFunc func()
}

// status returns the OCPP 2.0.1 runtime status
func (cs *CSMS20) status() []stationStatus {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	stations := []stationStatus{}

	for id, reg := range cs.regs {
		if id == "" {
			continue // skip anonymous registrations
		}

		state := StationStatusUnknown
		if cp := reg.cs; cp != nil {
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

	return stations
}

// SetUpdated sets a callback function that is called when the status changes
func (cs *CSMS20) SetUpdated(f func()) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.publishFunc = f
}

// errorHandler logs error channel
func (cs *CSMS20) errorHandler(errC <-chan error) {
	for err := range errC {
		cs.log.ERROR.Println(err)
	}
}

// publish triggers the publish callback if set
func (cs *CSMS20) publish() {
	if cs.publishFunc != nil {
		cs.publishFunc()
	}
}

func (cs *CSMS20) ChargingStationByID(id string) (*CS20CP, error) {
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

func (cs *CSMS20) WithEVSEStatus(id string, evseID int, fun func(status *availability.StatusNotificationRequest)) {
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
func (cs *CSMS20) RegisterChargingStation(id string, newfun func() *CS20CP, init func(*CS20CP) error) (*CS20CP, error) {
	cs.mu.Lock()

	// prepare shadow state
	reg, registered := cs.regs[id]
	if !registered {
		reg = newRegistration20()
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
func (cs *CSMS20) NewChargingStation(station ocpp2.ChargingStationConnection) {
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

		// update id
		cp.RegisterID(station.ID())
		cs.regs[station.ID()] = reg
		delete(cs.regs, "")

		cp.onTransportConnect()

		cs.mu.Unlock()
		cs.publish()
		return
	}

	// register unknown charging station
	cs.regs[station.ID()] = newRegistration20()
	cs.log.INFO.Printf("unknown charging station connected: %s", station.ID())

	cs.mu.Unlock()
	cs.publish()
}

// ChargingStationDisconnected implements ocpp2.ChargingStationConnectionHandler
func (cs *CSMS20) ChargingStationDisconnected(station ocpp2.ChargingStationConnection) {
	cs.log.DEBUG.Printf("charging station disconnected: %s", station.ID())

	if cp, err := cs.ChargingStationByID(station.ID()); err == nil {
		cp.connect(false)
	}

	cs.publish()
}

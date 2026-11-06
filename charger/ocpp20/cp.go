package ocpp20

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/charger/ocpp"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/provisioning"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/remotecontrol"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
)

// Station represents an OCPP 2.0.1 Charging Station
type Station struct {
	mu          sync.RWMutex
	log         *util.Logger
	onceConnect sync.Once
	onceMonitor sync.Once

	id string

	connected bool
	bootTimer *time.Timer // timeout for BootNotification wait after WebSocket connect
	connectC  chan struct{}
	meterC    chan struct{}

	// configuration properties
	PhaseSwitching          bool
	HasRemoteTriggerFeature bool
	ChargingRateUnit        types.ChargingRateUnitType
	ChargingProfileId       int
	StackLevel              int

	bootNotificationRequestC chan *provisioning.BootNotificationRequest
	BootNotificationResult   *provisioning.BootNotificationRequest

	evses map[evseKey]*EVSE
}

// evseKey identifies a registered EVSE by both EVSE id and connector id,
// so a single physical EVSE with multiple connectors can host multiple targets.
type evseKey struct {
	evseID, connectorID int
}

func NewStation(log *util.Logger, id string) *Station {
	return &Station{
		log: log,
		id:  id,

		evses: make(map[evseKey]*EVSE),

		connectC:                 make(chan struct{}, 1),
		meterC:                   make(chan struct{}, 1),
		bootNotificationRequestC: make(chan *provisioning.BootNotificationRequest, 1),

		ChargingRateUnit:        types.ChargingRateUnitAmperes,
		HasRemoteTriggerFeature: true, // assume remote trigger feature is available
	}
}

func (cp *Station) registerEVSE(evseID, connectorID int, evse *EVSE) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	k := evseKey{evseID, connectorID}
	if _, ok := cp.evses[k]; ok {
		return fmt.Errorf("evse already registered: %d/%d", evseID, connectorID)
	}

	cp.evses[k] = evse
	return nil
}

func (cp *Station) deregisterEVSE(evseID, connectorID int) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	delete(cp.evses, evseKey{evseID, connectorID})
}

// evsesForEvse returns all EVSE listeners registered for the given EVSE id,
// regardless of connector. Used for fan-out of EVSE-level events (e.g. MeterValues).
func (cp *Station) evsesForEvse(evseID int) []*EVSE {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	var out []*EVSE
	for k, evse := range cp.evses {
		if k.evseID == evseID {
			out = append(out, evse)
		}
	}
	return out
}

// evsesForConnector returns EVSE listeners that should receive an event for the
// given (evseID, connectorID). connectorID==0 means EVSE-wide; listeners
// registered with connectorID==0 always match, and listeners with positive
// connectorID match only their own connector.
func (cp *Station) evsesForConnector(evseID, connectorID int) []*EVSE {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	var out []*EVSE
	for k, evse := range cp.evses {
		if k.evseID != evseID {
			continue
		}
		if k.connectorID == 0 || connectorID == 0 || k.connectorID == connectorID {
			out = append(out, evse)
		}
	}
	return out
}

func (cp *Station) evseByTransactionID(id string) *EVSE {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	for _, evse := range cp.evses {
		if txn, err := evse.TransactionID(); err == nil && txn == id {
			return evse
		}
	}

	return nil
}

func (cp *Station) ID() string {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.id
}

func (cp *Station) RegisterID(id string) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.id == id {
		return nil
	}
	if cp.id != "" {
		return fmt.Errorf("ocpp: cannot re-register id %q (already %q)", id, cp.id)
	}

	cp.id = id
	return nil
}

// EnablePhaseSwitching forces phase switching on regardless of discovered capability.
// Used by manual config to override / supplement GetVariables auto-discovery.
func (cp *Station) EnablePhaseSwitching() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.PhaseSwitching = true
}

// stopBootTimer cancels and clears the boot notification wait timer.
// Must be called with cp.mu held.
func (cp *Station) stopBootTimer() {
	if cp.bootTimer != nil {
		cp.bootTimer.Stop()
		cp.bootTimer = nil
	}
}

func (cp *Station) connect(connect bool) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.connected = connect

	if connect {
		cp.onceConnect.Do(func() {
			close(cp.connectC)
		})
	} else {
		cp.stopBootTimer()
	}
}

// onTransportConnect is called when the WebSocket connection is established.
func (cp *Station) onTransportConnect() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.stopBootTimer()
	cp.bootTimer = time.AfterFunc(ocpp.Timeout, cp.onBootTimeout)

	// Proactively trigger BootNotification after a short delay.
	time.AfterFunc(ocpp.TriggerBootDelay, func() {
		cp.mu.RLock()
		if cp.bootTimer == nil || cp.BootNotificationResult != nil {
			cp.mu.RUnlock()
			return
		}
		cp.mu.RUnlock()

		cp.log.DEBUG.Printf("proactively triggering BootNotification")

		if err := Instance().TriggerMessage(
			cp.id,
			func(conf *remotecontrol.TriggerMessageResponse, err error) {
				if err != nil {
					cp.log.ERROR.Printf("trigger BootNotification response error: %v", err)
				}
			},
			remotecontrol.MessageTriggerBootNotification,
			func(request *remotecontrol.TriggerMessageRequest) {},
		); err != nil {
			cp.log.ERROR.Printf("failed to trigger BootNotification: %v", err)
		}
	})
}

// onBootTimeout is called when the BootNotification wait timer expires.
func (cp *Station) onBootTimeout() {
	cp.mu.Lock()
	if cp.bootTimer == nil {
		cp.mu.Unlock()
		return
	}
	cp.bootTimer = nil
	cp.mu.Unlock()

	cp.log.DEBUG.Printf("boot notification timeout, proceeding")
	cp.connect(true)
}

func (cp *Station) Connected() bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.connected
}

func (cp *Station) HasConnected() <-chan struct{} {
	return cp.connectC
}

// MonitorReboot ensures the given function runs only once per CP instance.
func (cp *Station) MonitorReboot(ctx context.Context, setup func() error) {
	cp.onceMonitor.Do(func() {
		// drain boot notification from initial setup
		select {
		case <-cp.bootNotificationRequestC:
		default:
		}

		go cp.monitorReboot(ctx, setup)
	})
}

func (cp *Station) monitorReboot(ctx context.Context, setup func() error) {
	for {
		select {
		case <-ctx.Done():
			return

		case boot := <-cp.bootNotificationRequestC:
			cp.log.INFO.Printf("reboot detected (model: %s, vendor: %s), re-initializing",
				boot.ChargingStation.Model, boot.ChargingStation.VendorName)

			if err := setup(); err != nil {
				cp.log.ERROR.Printf("failed to re-initialize after reboot: %v", err)
			}
		}
	}
}

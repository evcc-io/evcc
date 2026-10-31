package ocpp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// Since ocpp-go interfaces at charge point level, we need to manage multiple connector separately

type CP struct {
	mu          sync.RWMutex
	cs          *CS // central system this charge point is registered with
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
	NumberOfConnectors      int
	IdTag                   string

	meterValuesSample        string
	bootNotificationRequestC chan *core.BootNotificationRequest
	BootNotificationResult   *core.BootNotificationRequest

	connectors map[int]*Connector
}

func NewChargePoint(log *util.Logger, cs *CS, id string) *CP {
	return &CP{
		cs:  cs,
		log: log,
		id:  id,

		connectors: make(map[int]*Connector),

		connectC:                 make(chan struct{}, 1),
		meterC:                   make(chan struct{}, 1),
		bootNotificationRequestC: make(chan *core.BootNotificationRequest, 1),

		ChargingRateUnit:        "A",
		HasRemoteTriggerFeature: true, // assume remote trigger feature is available
	}
}

func (cp *CP) registerConnector(id int, conn *Connector) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if _, ok := cp.connectors[id]; ok {
		return fmt.Errorf("connector already registered: %d", id)
	}

	cp.connectors[id] = conn
	return nil
}

func (cp *CP) deregisterConnector(id int) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	delete(cp.connectors, id)
}

func (cp *CP) connectorByID(id int) *Connector {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.connectors[id]
}

// markRebooted flags all connectors after a BootNotification.
func (cp *CP) markRebooted() {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	for _, conn := range cp.connectors {
		conn.markRebooted()
	}
}

func (cp *CP) connectorByTransactionID(id int) *Connector {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	for _, conn := range cp.connectors {
		if txn, err := conn.TransactionID(); err == nil && txn == id {
			return conn
		}
	}

	return nil
}

func (cp *CP) ID() string {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.id
}

func (cp *CP) RegisterID(id string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.id != "" {
		panic("ocpp: cannot re-register id")
	}

	cp.id = id
}

// stopBootTimer cancels and clears the boot notification wait timer.
// Must be called with cp.mu held.
func (cp *CP) stopBootTimer() {
	if cp.bootTimer != nil {
		cp.bootTimer.Stop()
		cp.bootTimer = nil
	}
}

func (cp *CP) connect(connect bool) {
	cp.mu.Lock()
	changed := cp.setConnected(connect)
	cp.mu.Unlock()

	cp.publishStatus(changed)
}

// setConnected updates the connection state and reports whether it changed.
// Must be called with cp.mu held.
func (cp *CP) setConnected(connect bool) bool {
	changed := cp.connected != connect
	cp.connected = connect

	if connect {
		cp.onceConnect.Do(func() {
			close(cp.connectC)
		})
	} else {
		cp.stopBootTimer()
	}

	return changed
}

// publishStatus must be called without cp.mu held, the status callback reads Connected().
func (cp *CP) publishStatus(changed bool) {
	if changed && cp.cs != nil {
		cp.cs.publish()
	}
}

// onTransportConnect is called when the WebSocket connection is established.
// Instead of marking the CP as connected immediately, it waits for the
// BootNotification handshake to complete (or a timeout to expire).
//
// Some chargers (e.g. Wallbox Pulsar Pro FW 6.x) do not send BootNotification
// spontaneously on connect. For these chargers, we proactively trigger it
// after a short delay, which typically yields a response within 1 second
// instead of waiting for the full timeout.
func (cp *CP) onTransportConnect() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.stopBootTimer()
	// timer is passed by reference since the callback may fire before the assignment
	var timer *time.Timer
	timer = time.AfterFunc(Timeout, func() {
		cp.endBootWait(&timer, "boot notification timeout")
	})
	cp.bootTimer = timer

	// Proactively trigger BootNotification after a short delay.
	// This helps chargers that don't send it spontaneously (e.g. Wallbox FW 6.x).
	// The TriggerMessage is sent directly via the OCPP instance, bypassing the
	// Connected() check which would fail at this point.
	time.AfterFunc(TriggerBootDelay, func() {
		cp.mu.RLock()
		// BootNotification arrived, disconnect or newer connection: the timer has been replaced
		if cp.bootTimer != timer {
			cp.mu.RUnlock()
			return
		}
		cp.mu.RUnlock()

		cp.log.DEBUG.Printf("proactively triggering BootNotification")

		if err := cp.cs.TriggerMessage(
			cp.id,
			func(conf *remotetrigger.TriggerMessageConfirmation, err error) {
				if err != nil {
					cp.log.ERROR.Printf("trigger BootNotification response error: %v", err)
					return
				}
				if conf != nil && conf.Status != remotetrigger.TriggerMessageStatusAccepted {
					cp.endBootWait(&timer, "trigger BootNotification: "+string(conf.Status))
				}
			},
			core.BootNotificationFeatureName,
			func(request *remotetrigger.TriggerMessageRequest) {},
		); err != nil {
			cp.log.ERROR.Printf("failed to trigger BootNotification: %v", err)
		}
	})
}

// endBootWait stops waiting for BootNotification if timer still belongs to the current connection.
func (cp *CP) endBootWait(timer **time.Timer, reason string) {
	cp.mu.Lock()
	if cp.bootTimer != *timer {
		// BootNotification arrived, disconnect or newer connection
		cp.mu.Unlock()
		return
	}

	cp.stopBootTimer()
	changed := cp.setConnected(true)
	cp.mu.Unlock()

	cp.log.DEBUG.Printf("%s, proceeding", reason)
	cp.publishStatus(changed)
}

func (cp *CP) Connected() bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.connected
}

func (cp *CP) HasConnected() <-chan struct{} {
	return cp.connectC
}

// MonitorReboot ensures the given function runs only once per CP instance.
// Used to start the reboot monitor goroutine for multi-connector charge points.
func (cp *CP) MonitorReboot(ctx context.Context, setup func() error) {
	cp.onceMonitor.Do(func() {
		// drain boot notification from initial setup
		select {
		case <-cp.bootNotificationRequestC:
		default:
		}

		go cp.monitorReboot(ctx, setup)
	})
}

func (cp *CP) monitorReboot(ctx context.Context, setup func() error) {
	for {
		select {
		case <-ctx.Done():
			return

		case boot := <-cp.bootNotificationRequestC:
			cp.log.INFO.Printf("reboot detected (model: %s, vendor: %s), re-initializing",
				boot.ChargePointModel, boot.ChargePointVendor)

			if err := setup(); err != nil {
				cp.log.ERROR.Printf("failed to re-initialize after reboot: %v", err)
			}
		}
	}
}

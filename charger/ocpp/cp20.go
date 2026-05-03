package ocpp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/provisioning"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/remotecontrol"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
)

// CS20CP represents an OCPP 2.0.1 Charging Station
type CS20CP struct {
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

	evses map[int]*EVSE
}

func NewCS20CP(log *util.Logger, id string) *CS20CP {
	return &CS20CP{
		log: log,
		id:  id,

		evses: make(map[int]*EVSE),

		connectC:                 make(chan struct{}, 1),
		meterC:                   make(chan struct{}, 1),
		bootNotificationRequestC: make(chan *provisioning.BootNotificationRequest, 1),

		ChargingRateUnit:        types.ChargingRateUnitAmperes,
		HasRemoteTriggerFeature: true, // assume remote trigger feature is available
	}
}

func (cp *CS20CP) registerEVSE(id int, evse *EVSE) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if _, ok := cp.evses[id]; ok {
		return fmt.Errorf("evse already registered: %d", id)
	}

	cp.evses[id] = evse
	return nil
}

func (cp *CS20CP) evseByID(id int) *EVSE {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.evses[id]
}

func (cp *CS20CP) evseByTransactionID(id string) *EVSE {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	for _, evse := range cp.evses {
		if txn, err := evse.TransactionID(); err == nil && txn == id {
			return evse
		}
	}

	return nil
}

func (cp *CS20CP) ID() string {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.id
}

func (cp *CS20CP) RegisterID(id string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.id != "" {
		panic("ocpp: cannot re-register id")
	}

	cp.id = id
}

// stopBootTimer cancels and clears the boot notification wait timer.
// Must be called with cp.mu held.
func (cp *CS20CP) stopBootTimer() {
	if cp.bootTimer != nil {
		cp.bootTimer.Stop()
		cp.bootTimer = nil
	}
}

func (cp *CS20CP) connect(connect bool) {
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
func (cp *CS20CP) onTransportConnect() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.stopBootTimer()
	cp.bootTimer = time.AfterFunc(Timeout, cp.onBootTimeout)

	// Proactively trigger BootNotification after a short delay.
	time.AfterFunc(triggerBootDelay, func() {
		cp.mu.RLock()
		if cp.bootTimer == nil || cp.BootNotificationResult != nil {
			cp.mu.RUnlock()
			return
		}
		cp.mu.RUnlock()

		cp.log.DEBUG.Printf("proactively triggering BootNotification")

		if err := Instance20().TriggerMessage(
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
func (cp *CS20CP) onBootTimeout() {
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

func (cp *CS20CP) Connected() bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.connected
}

func (cp *CS20CP) HasConnected() <-chan struct{} {
	return cp.connectC
}

// MonitorReboot ensures the given function runs only once per CP instance.
func (cp *CS20CP) MonitorReboot(ctx context.Context, setup func() error) {
	cp.onceMonitor.Do(func() {
		// drain boot notification from initial setup
		select {
		case <-cp.bootNotificationRequestC:
		default:
		}

		go cp.monitorReboot(ctx, setup)
	})
}

func (cp *CS20CP) monitorReboot(ctx context.Context, setup func() error) {
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

package ocpp

import (
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// Since ocpp-go interfaces at charge point level, we need to manage multiple connector separately

type CP struct {
	mu          sync.RWMutex
	log         *util.Logger
	onceConnect sync.Once
	onceMonitor sync.Once

	id string

	connected   bool
	initialized bool        // true after first Setup completes
	bootTimer   *time.Timer // timeout for BootNotification wait after WebSocket connect
	connectC    chan struct{}
	meterC      chan struct{}

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

func NewChargePoint(log *util.Logger, id string) *CP {
	return &CP{
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

func (cp *CP) connect(connect bool) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.connected = connect

	if connect {
		cp.onceConnect.Do(func() {
			close(cp.connectC)
		})
	} else {
		// cancel boot timer on disconnect
		if cp.bootTimer != nil {
			cp.bootTimer.Stop()
			cp.bootTimer = nil
		}
	}
}

// onTransportConnect is called when the WebSocket connection is established.
// Instead of marking the CP as connected immediately, it waits for the
// BootNotification handshake to complete (or a timeout to expire).
func (cp *CP) onTransportConnect() {
	cp.mu.Lock()

	// cancel any previous boot timer
	if cp.bootTimer != nil {
		cp.bootTimer.Stop()
	}

	cp.bootTimer = time.AfterFunc(Timeout, cp.onBootTimeout)

	cp.mu.Unlock()
}

// onBootTimeout is called when the BootNotification wait timer expires.
func (cp *CP) onBootTimeout() {
	cp.mu.Lock()
	cp.bootTimer = nil
	connected := cp.connected
	cp.mu.Unlock()

	if !connected {
		cp.log.DEBUG.Printf("boot notification timeout, proceeding")
		cp.connect(true)
	}
}

func (cp *CP) Connected() bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.connected
}

func (cp *CP) Initialized() bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.initialized
}

func (cp *CP) HasConnected() <-chan struct{} {
	return cp.connectC
}

// BootNotificationC returns the channel for monitoring BootNotification messages.
func (cp *CP) BootNotificationC() <-chan *core.BootNotificationRequest {
	return cp.bootNotificationRequestC
}

// StartMonitor ensures the given function runs only once per CP instance.
// Used to start the reboot monitor goroutine for multi-connector charge points.
func (cp *CP) StartMonitor(start func()) {
	cp.onceMonitor.Do(start)
}

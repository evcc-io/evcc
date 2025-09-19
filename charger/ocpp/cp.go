package ocpp

import (
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

// Since ocpp-go interfaces at charge point level, we need to manage multiple connector separately

type CP struct {
	mu          sync.RWMutex
	log         *util.Logger
	onceConnect sync.Once
	onceBoot    sync.Once

	id string

	connected bool
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
	}
}

func (cp *CP) Connected() bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.connected
}

func (cp *CP) HasConnected() <-chan struct{} {
	return cp.connectC
}

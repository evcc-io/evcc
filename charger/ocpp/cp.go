package ocpp

import (
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/util"
)

// TODO support multiple connectors
// Since ocpp-go interfaces at charge point level, we need to manage multiple connector separately

type CP struct {
	mu   sync.Mutex
	once sync.Once
	log  *util.Logger

	id string

	connected bool
	connectC  chan struct{}

	connectors map[int]*Connector
}

func NewChargePoint(log *util.Logger, id string) *CP {
	return &CP{
		log: log,
		id:  id,

		connectC:   make(chan struct{}),
		connectors: make(map[int]*Connector),
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

func (cp *CP) connectorByID(id int) *Connector {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	return cp.connectors[id]
}

func (cp *CP) connectorByTransactionID(id int) *Connector {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	for _, conn := range cp.connectors {
		if txn, err := conn.TransactionID(); err == nil && txn == id {
			return conn
		}
	}

	return nil
}

func (cp *CP) ID() string {
	cp.mu.Lock()
	defer cp.mu.Unlock()

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
		cp.once.Do(func() {
			close(cp.connectC)
		})
	}
}

func (cp *CP) Connected() bool {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	return cp.connected
}

func (cp *CP) HasConnected() <-chan struct{} {
	return cp.connectC
}

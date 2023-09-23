package ocpp

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
)

var mu sync.Mutex

type ChargePointConnector struct {
	*CP
	*Connector
}

func NewChargePointConnector(log *util.Logger, id string, connector int, timeout time.Duration) *ChargePointConnector {
	mu.Lock()
	defer mu.Unlock()

	cp, err := Instance().chargepointByID(id)
	if err != nil {
		cp = NewChargePoint(log, id, timeout)
	}

	return &ChargePointConnector{
		CP:        cp,
		Connector: NewConnector(log, connector),
	}
}

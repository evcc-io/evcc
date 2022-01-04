package ocpp

import (
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

type CP struct {
	mu          sync.Mutex
	log         *util.Logger
	id          string
	available   bool
	meterValues []types.MeterValue
}

func (cp *CP) SetAvailable(available bool) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.available = available
}

func (cp *CP) Available() bool {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	return cp.available
}

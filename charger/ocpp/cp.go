package ocpp

import (
	"sync"
)

type CP struct {
	mu        sync.Mutex
	id        string
	available bool
}

func (cp *CP) SetAvailable(available bool) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.available = available
}

package plugin

import (
	"sync"
	"sync/atomic"
	"time"
)

type availabilityHandler struct {
	asExpected      atomic.Bool
	expectedPayload string
	ready           chan struct{}
	once            sync.Once
}

func (h *availabilityHandler) receive(payload string) {
	h.asExpected.Store(payload == h.expectedPayload)

	h.once.Do(func() {
		close(h.ready)
	})
}

func (h *availabilityHandler) AsExpected() bool {
	return h.asExpected.Load()
}

func (h *availabilityHandler) Wait(timeout time.Duration) bool {
	select {
	case <-h.ready:
		return true
	case <-time.After(timeout):
		return false
	}
}

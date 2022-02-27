package core

import (
	"sync/atomic"
	"time"
)

// Health is a health checker that needs regular updates to stay healthy
type Health struct {
	locker  uint32 // mutex
	updated time.Time
	timeout time.Duration
}

// NewHealth creates new health checker
func NewHealth(timeout time.Duration) (health *Health) {
	return &Health{timeout: timeout}
}

// Healthy returns health status based on last update timestamp
func (health *Health) Healthy() bool {
	if health == nil {
		return false
	}

	start := time.Now()

	for time.Since(start) < time.Second {
		if atomic.CompareAndSwapUint32(&health.locker, 0, 1) {
			defer atomic.StoreUint32(&health.locker, 0)
			return time.Since(health.updated) < health.timeout
		}

		time.Sleep(50 * time.Millisecond)
	}

	return false
}

// Update updates the health timer on each loadpoint update
func (health *Health) Update() {
	if health == nil {
		return
	}

	start := time.Now()

	for time.Since(start) < time.Second {
		if atomic.CompareAndSwapUint32(&health.locker, 0, 1) {
			health.updated = time.Now()
			atomic.StoreUint32(&health.locker, 0)
			return
		}

		time.Sleep(50 * time.Millisecond)
	}
}

package core

import (
	"sync"
	"time"
)

// Health is a health checker that needs regular updates to stay healthy
type Health struct {
	mux     sync.Mutex
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

	health.mux.Lock()
	defer health.mux.Unlock()

	return time.Since(health.updated) < health.timeout
}

// Update updates the health timer on each loadpoint update
func (health *Health) Update() {
	if health == nil {
		return
	}

	health.mux.Lock()
	defer health.mux.Unlock()

	health.updated = time.Now()
}

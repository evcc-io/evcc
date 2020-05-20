package util

import (
	"sync"
	"time"
)

const waitTimeout = 50 * time.Millisecond // polling interval when waiting for initial value

// Waiter provides monitoring of receive timeouts and reception of initial value
type Waiter struct {
	sync.Mutex
	log     func()
	once    sync.Once
	updated time.Time
	timeout time.Duration
}

// NewWaiter creates new waiter
func NewWaiter(timeout time.Duration, logInitialWait func()) *Waiter {
	return &Waiter{
		log:     logInitialWait,
		timeout: timeout,
	}
}

// Update is called when client has received data. Update resets the timeout counter.
// It is client responsibility to ensure that the waiter is not locked when Update is called.
func (p *Waiter) Update() {
	p.updated = time.Now()
}

// waitForInitialValue blocks until Update has been called at least once.
// It assumes lock has been obtained before and returns with lock active.
func (p *Waiter) waitForInitialValue() {
	if p.updated.IsZero() {
		p.log()

		// wait for initial update
		for p.updated.IsZero() {
			p.Unlock()
			time.Sleep(waitTimeout)
			p.Lock()
		}
	}
}

// LockWithTimeout waits for initial value and checks if update timeout has elapsed
func (p *Waiter) LockWithTimeout() time.Duration {
	p.Lock()

	// waiting assumes lock acquired and returns with lock
	p.once.Do(p.waitForInitialValue)

	if elapsed := time.Since(p.updated); p.timeout != 0 && elapsed > p.timeout {
		return elapsed
	}

	return 0
}

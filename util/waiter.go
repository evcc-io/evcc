package util

import (
	"sync"
	"time"
)

var waitInitialTimeout = 10 * time.Second

// Waiter provides monitoring of receive timeouts and reception of initial value
type Waiter struct {
	mu      sync.Mutex
	log     func()
	updated time.Time
	timeout time.Duration
	initial chan bool
}

// NewWaiter creates new waiter
func NewWaiter(timeout time.Duration, logInitialWait func()) *Waiter {
	return &Waiter{
		log:     logInitialWait,
		timeout: timeout,
		initial: make(chan bool),
	}
}

// Update is called when client has received data. Update resets the timeout counter.
func (p *Waiter) Update() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.updated = time.Now()

	select {
	case <-p.initial:
	default:
		close(p.initial)
	}
}

// Overdue waits for initial update and returns the duration since the last update
// in excess of timeout.
func (p *Waiter) Overdue() time.Duration {
	select {
	case <-p.initial:
	default:
		p.log()

		select {
		case <-p.initial:
		case <-time.After(waitInitialTimeout):
			return waitInitialTimeout
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if elapsed := time.Since(p.updated); p.timeout != 0 && elapsed > p.timeout {
		return elapsed
	}

	return 0
}

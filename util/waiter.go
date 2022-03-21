package util

import (
	"sync"
	"time"
)

var waitInitialTimeout = 10 * time.Second

// Waiter provides monitoring of receive timeouts and reception of initial value
type Waiter struct {
	sync.Mutex
	log     func()
	cond    *sync.Cond
	updated time.Time
	timeout time.Duration
}

// NewWaiter creates new waiter
func NewWaiter(timeout time.Duration, logInitialWait func()) *Waiter {
	p := &Waiter{
		log:     logInitialWait,
		timeout: timeout,
	}
	p.cond = sync.NewCond(p)
	return p
}

// Update is called when client has received data. Update resets the timeout counter.
// Waiter MUST be locked when calling Update.
func (p *Waiter) Update() {
	p.updated = time.Now()
	p.cond.Broadcast()
}

// Overdue waits for initial update and returns the duration since the last update
// in excess of timeout. Waiter MUST be locked when calling Overdue.
func (p *Waiter) Overdue() time.Duration {
	if p.updated.IsZero() {
		p.log()

		c := make(chan struct{})

		go func() {
			defer close(c)
			p.Lock() // establish lock once go routine has started
			for p.updated.IsZero() {
				p.cond.Wait()
			}
		}()

		// release lock so external updates can occur
		p.Unlock()

		select {
		case <-c:
			// initial value received, lock established
		case <-time.After(waitInitialTimeout):
			// establish lock per contract of `Update()``
			p.Lock()
			p.Update() // unblock the sync.Cond
			p.Unlock()
			<-c                     // wait for goroutine, re-establish lock
			p.updated = time.Time{} // reset to "initial value missing"
			return waitInitialTimeout
		}
	}

	if elapsed := time.Since(p.updated); p.timeout != 0 && elapsed > p.timeout {
		return elapsed
	}

	return 0
}

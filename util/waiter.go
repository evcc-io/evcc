package util

import (
	"sync"
	"time"
)

var WaitInitialTimeout = 10 * time.Second

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
// It is client responsibility to ensure that the waiter is not locked when Update is called.
func (p *Waiter) Update() {
	p.updated = time.Now()
	p.cond.Broadcast()
}

// LockWithTimeout waits for initial value and checks if update timeout has elapsed
func (p *Waiter) LockWithTimeout() time.Duration {
	p.Lock()

	if p.updated.IsZero() {
		p.log()

		c := make(chan struct{})

		go func() {
			defer close(c)
			for p.updated.IsZero() {
				p.cond.Wait()
			}
		}()

		select {
		case <-c:
			// initial value received, lock established
		case <-time.After(WaitInitialTimeout):
			p.Update()              // unblock the sync.Cond
			<-c                     // wait for goroutine, re-establish lock
			p.updated = time.Time{} // reset updated to missing initial value
			return WaitInitialTimeout
		}
	}

	if elapsed := time.Since(p.updated); p.timeout != 0 && elapsed > p.timeout {
		return elapsed
	}

	return 0
}

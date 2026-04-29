package remote

import (
	"sync"
	"time"
)

// authRateLimiter tracks failed authentication attempts in a sliding window.
// When the failure count exceeds the threshold, further attempts are blocked
// to prevent brute-force attacks.
type authRateLimiter struct {
	mu       sync.Mutex
	failures []time.Time
	window   time.Duration
	max      int
	now      func() time.Time
}

func newAuthRateLimiter() *authRateLimiter {
	return &authRateLimiter{
		window: time.Minute,
		max:    10,
		now:    time.Now,
	}
}

// allow checks whether an authentication attempt should proceed.
func (rl *authRateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := rl.now().Add(-rl.window)

	// prune old entries
	valid := rl.failures[:0]
	for _, t := range rl.failures {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	rl.failures = valid

	return len(rl.failures) < rl.max
}

// fail records a failed authentication attempt.
func (rl *authRateLimiter) fail() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.failures = append(rl.failures, rl.now())
}

package util

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
)

type Monitor[T any] struct {
	val     T
	mu      sync.RWMutex
	once    sync.Once
	done    chan struct{}
	updated time.Time
	timeout time.Duration
}

func NewMonitor[T any](timeout time.Duration) *Monitor[T] {
	return &Monitor[T]{
		done:    make(chan struct{}),
		timeout: timeout,
	}
}

// Set updates the current value and timestamp
func (m *Monitor[T]) Set(val T) {
	m.SetFunc(func(v *T) { *v = val })
}

// SetFunc updates the current value and timestamp while holding the lock
func (m *Monitor[T]) SetFunc(set func(*T)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	set(&m.val)
	m.updated = time.Now()

	m.once.Do(func() { close(m.done) })
}

// Get returns the current value or ErrOutdated if timeout exceeded
func (m *Monitor[T]) Get() (T, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// without timeout set, error if not yet received
	if m.timeout == 0 {
		select {
		case <-m.done:
			return m.val, nil
		default:
			return m.val, api.ErrOutdated
		}
	} else if time.Since(m.updated) > m.timeout {
		return m.val, api.ErrOutdated
	}

	return m.val, nil
}

// Done signals if monitor has been updated at least once
func (m *Monitor[T]) Done() <-chan struct{} {
	return m.done
}

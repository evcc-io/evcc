package util

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
)

// Monitor monitors values for regular updates
type Monitor[T any] struct {
	val     T
	mu      sync.RWMutex
	once    sync.Once
	done    chan struct{}
	updated time.Time
	timeout time.Duration
}

// NewMonitor created a new monitor with given timeout
func NewMonitor[T any](timeout time.Duration) *Monitor[T] {
	return &Monitor[T]{
		done:    make(chan struct{}),
		timeout: timeout,
	}
}

// Set updates the current value and timestamp
func (m *Monitor[T]) Set(val T) {
	m.SetFunc(func(_ T) T { return val })
}

// SetFunc updates the current value and timestamp while holding the lock
func (m *Monitor[T]) SetFunc(set func(T) T) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.val = set(m.val)
	m.updated = time.Now()

	m.once.Do(func() { close(m.done) })
}

// Get returns the current value or ErrOutdated if timeout exceeded
func (m *Monitor[T]) Get() (T, error) {
	var res T
	err := m.GetFunc(func(v T) {
		res = v
	})
	return res, err
}

// GetFunc returns the current value or ErrOutdated if timeout exceeded while holding the lock
func (m *Monitor[T]) GetFunc(get func(T)) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// without timeout set, error if not yet received
	if m.timeout == 0 {
		select {
		case <-m.done:
			get(m.val)
			return nil
		default:
			return api.ErrOutdated
		}
	} else if time.Since(m.updated) > m.timeout {
		get(m.val)
		return api.ErrOutdated
	}

	get(m.val)
	return nil
}

// Done signals if monitor has been updated at least once
func (m *Monitor[T]) Done() <-chan struct{} {
	return m.done
}

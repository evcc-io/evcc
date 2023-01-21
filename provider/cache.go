package provider

import (
	"errors"
	"sync"
	"time"

	"github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

var (
	bus = EventBus.New()
	log = util.NewLogger("cache")
)

const reset = "reset"

func ResetCached() {
	bus.Publish(reset)
}

// cached wraps a getter with a cache
type cached[T any] struct {
	mux     sync.Mutex
	clock   clock.Clock
	updated time.Time
	cache   time.Duration
	g       func() (T, error)
	val     T
	err     error
}

// Cached wraps a getter with a cache
func Cached[T any](g func() (T, error), cache time.Duration) func() (T, error) {
	c := ResettableCached(g, cache)
	return c.Get
}

// Cacheable is the interface for a resettable cache
type Cacheable[T any] interface {
	Get() (T, error)
	Reset()
}

var _ Cacheable[int64] = (*cached[int64])(nil)

// ResettableCached wraps a getter with a cache. It returns a `Cacheable`.
// Instead of the cached getter, the `Get()` and `Reset()` methods are exposed.
func ResettableCached[T any](g func() (T, error), cache time.Duration) *cached[T] {
	c := &cached[T]{
		clock: clock.New(),
		cache: cache,
		g:     g,
	}
	_ = bus.Subscribe(reset, c.Reset)
	return c
}

func (c *cached[T]) Get() (T, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.mustUpdate() {
		c.val, c.err = c.g()
		c.updated = c.clock.Now()
	}

	return c.val, c.err
}

func (c *cached[T]) Reset() {
	c.mux.Lock()
	c.updated = time.Time{}
	c.mux.Unlock()
}

func (c *cached[T]) mustUpdate() bool {
	return c.clock.Since(c.updated) > c.cache || errors.Is(c.err, api.ErrMustRetry)
}

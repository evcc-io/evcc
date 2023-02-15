package provider

import (
	"errors"
	"math"
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
	mux              sync.Mutex
	clock            clock.Clock
	updated          time.Time
	retried          time.Time
	cache            time.Duration
	retryWithBackoff bool
	backoffCounter   int
	g                func() (T, error)
	val              T
	err              error
}

// Cached wraps a getter with a cache
func Cached[T any](g func() (T, error), cache time.Duration, retryWithBackoff bool) func() (T, error) {
	c := ResettableCached(g, cache, retryWithBackoff)
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
func ResettableCached[T any](g func() (T, error), cache time.Duration, retryWithBackoff bool) *cached[T] {
	c := &cached[T]{
		clock:            clock.New(),
		cache:            cache,
		g:                g,
		retryWithBackoff: retryWithBackoff,
		retried:          clock.New().Now(),
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
		c.retried = c.clock.Now()
	}

	if c.err == nil {
		c.resetBackoff()
	}

	return c.val, c.err
}

func (c *cached[T]) Reset() {
	c.mux.Lock()
	c.updated = time.Time{}
	c.retried = time.Time{}
	c.mux.Unlock()
}

func (c *cached[T]) mustUpdate() bool {
	return (c.clock.Since(c.updated) > c.cache && c.err == nil) || errors.Is(c.err, api.ErrMustRetry) || c.shouldRetryWithBackoff(c.err)
}

func (c *cached[T]) shouldRetryWithBackoff(err error) bool {
	if err != nil && c.retryWithBackoff {
		// Exponentially backoff for 2^n minutes. Maximum backoff wait time is regular cache time
		waitTime := time.Duration(math.Min(math.Pow(2, float64(c.backoffCounter)), c.cache.Seconds())*60) * time.Second

		if c.clock.Since(c.retried) > waitTime {
			log.DEBUG.Printf("Retrying cache get after error with backoff wait time %v", waitTime)

			c.retried = c.clock.Now()
			c.backoffCounter++
			return true
		}
	}

	return false
}

func (c *cached[T]) resetBackoff() {
	log.DEBUG.Printf("Resetting backoff")
	c.backoffCounter = 0
	c.retried = c.clock.Now()
}

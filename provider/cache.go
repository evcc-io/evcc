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

// Cached wraps a getter with a cache
type Cached[T any] struct {
	mux     sync.Mutex
	clock   clock.Clock
	updated time.Time
	cache   time.Duration
	getter  func()(T,error)
	val     T
	err     error
}

// NewCached wraps a getter with a cache
func NewCached[T any](getter func()(T,error), cache time.Duration) *Cached[T] {
	c := &Cached[T]{
		clock:  clock.New(),
		getter: getter,
		cache:  cache,
	}

	_ = bus.Subscribe(reset, c.reset)

	return c
}

func (c *Cached[T]) reset() {
	c.mux.Lock()
	c.updated = time.Time{}
	c.mux.Unlock()
}

func (c *Cached[T]) mustUpdate() bool {
	return c.clock.Since(c.updated) > c.cache || errors.Is(c.err, api.ErrMustRetry)
}

// Get returns cached value, refreshing as necessary
func (c *Cached[T]) Get()(T, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.mustUpdate() {
		c.val, c.err = c.getter()
		c.updated = c.clock.Now()
	}

	return c.val, c.err
}

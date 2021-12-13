package provider

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/logx"
)

var (
	bus = EventBus.New()
	log = logx.NewModule("cache")
)

const reset = "reset"

func ResetCached() {
	bus.Publish(reset)
}

// Cached wraps a getter with a cache
type Cached struct {
	mux     sync.Mutex
	clock   clock.Clock
	updated time.Time
	cache   time.Duration
	getter  interface{}
	val     interface{}
	err     error
}

// NewCached wraps a getter with a cache
func NewCached(getter interface{}, cache time.Duration) *Cached {
	c := &Cached{
		clock:  clock.New(),
		getter: getter,
		cache:  cache,
	}

	_ = bus.Subscribe(reset, c.reset)

	return c
}

func fatal(i interface{}) {
	logx.Error(log, "error", "invalid type: %T", i)
	os.Exit(1)
}

func (c *Cached) reset() {
	c.mux.Lock()
	c.updated = time.Time{}
	c.mux.Unlock()
}

func (c *Cached) mustUpdate() bool {
	return c.clock.Since(c.updated) > c.cache || errors.Is(c.err, api.ErrMustRetry)
}

// FloatGetter gets float value
func (c *Cached) FloatGetter() func() (float64, error) {
	g, ok := c.getter.(func() (float64, error))
	if !ok {
		fatal(c.getter)
	}

	return func() (float64, error) {
		c.mux.Lock()
		defer c.mux.Unlock()

		if c.mustUpdate() {
			c.val, c.err = g()
			c.updated = c.clock.Now()
		}

		return c.val.(float64), c.err
	}
}

// IntGetter gets int value
func (c *Cached) IntGetter() func() (int64, error) {
	g, ok := c.getter.(func() (int64, error))
	if !ok {
		fatal(c.getter)
	}

	return func() (int64, error) {
		c.mux.Lock()
		defer c.mux.Unlock()

		if c.mustUpdate() {
			c.val, c.err = g()
			c.updated = c.clock.Now()
		}

		return c.val.(int64), c.err
	}
}

// StringGetter gets string value
func (c *Cached) StringGetter() func() (string, error) {
	g, ok := c.getter.(func() (string, error))
	if !ok {
		fatal(c.getter)
	}

	return func() (string, error) {
		c.mux.Lock()
		defer c.mux.Unlock()

		if c.mustUpdate() {
			c.val, c.err = g()
			c.updated = c.clock.Now()
		}

		return c.val.(string), c.err
	}
}

// BoolGetter gets bool value
func (c *Cached) BoolGetter() func() (bool, error) {
	g, ok := c.getter.(func() (bool, error))
	if !ok {
		fatal(c.getter)
	}

	return func() (bool, error) {
		c.mux.Lock()
		defer c.mux.Unlock()

		if c.mustUpdate() {
			c.val, c.err = g()
			c.updated = c.clock.Now()
		}

		return c.val.(bool), c.err
	}
}

// DurationGetter gets time.Duration value
func (c *Cached) DurationGetter() func() (time.Duration, error) {
	g, ok := c.getter.(func() (time.Duration, error))
	if !ok {
		fatal(c.getter)
	}

	return func() (time.Duration, error) {
		c.mux.Lock()
		defer c.mux.Unlock()

		if c.mustUpdate() {
			c.val, c.err = g()
			c.updated = c.clock.Now()
		}

		return c.val.(time.Duration), c.err
	}
}

// TimeGetter gets time.Time value
func (c *Cached) TimeGetter() func() (time.Time, error) {
	g, ok := c.getter.(func() (time.Time, error))
	if !ok {
		fatal(c.getter)
	}

	return func() (time.Time, error) {
		c.mux.Lock()
		defer c.mux.Unlock()

		if c.mustUpdate() {
			c.val, c.err = g()
			c.updated = c.clock.Now()
		}

		return c.val.(time.Time), c.err
	}
}

// InterfaceGetter gets interface value
func (c *Cached) InterfaceGetter() func() (interface{}, error) {
	g, ok := c.getter.(func() (interface{}, error))
	if !ok {
		fatal(c.getter)
	}

	return func() (interface{}, error) {
		c.mux.Lock()
		defer c.mux.Unlock()

		if c.mustUpdate() {
			c.val, c.err = g()
			c.updated = c.clock.Now()
		}

		return c.val, c.err
	}
}

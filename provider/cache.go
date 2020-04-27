package provider

import (
	"time"

	"github.com/andig/evcc/util"
	"github.com/benbjohnson/clock"
)

// Cached wraps a getter with a cache
type Cached struct {
	log     *util.Logger
	clock   clock.Clock
	updated time.Time
	cache   time.Duration
	getter  interface{}
	val     interface{}
}

// NewCached wraps a getter with a cache
func NewCached(log *util.Logger, getter interface{}, cache time.Duration) *Cached {
	return &Cached{
		log:    log,
		clock:  clock.New(),
		getter: getter,
		cache:  cache,
	}
}

// FloatGetter gets float value
func (c *Cached) FloatGetter() FloatGetter {
	g, ok := c.getter.(FloatGetter)
	if !ok {
		if g, ok = c.getter.(func() (float64, error)); !ok {
			c.log.FATAL.Fatalf("invalid type: %T", c.getter)
		}
		g = FloatGetter(g)
	}

	return FloatGetter(func() (float64, error) {
		if c.clock.Since(c.updated) > c.cache {
			val, err := g()
			if err != nil {
				return val, err
			}

			c.updated = c.clock.Now()
			c.val = val
		}

		return c.val.(float64), nil
	})
}

// IntGetter gets int value
func (c *Cached) IntGetter() IntGetter {
	g, ok := c.getter.(IntGetter)
	if !ok {
		if g, ok = c.getter.(func() (int64, error)); !ok {
			c.log.FATAL.Fatalf("invalid type: %T", c.getter)
		}
		g = IntGetter(g)
	}

	return IntGetter(func() (int64, error) {
		if c.clock.Since(c.updated) > c.cache {
			val, err := g()
			if err != nil {
				return val, err
			}

			c.updated = c.clock.Now()
			c.val = val
		}

		return c.val.(int64), nil
	})
}

// StringGetter gets string value
func (c *Cached) StringGetter() StringGetter {
	g, ok := c.getter.(StringGetter)
	if !ok {
		if g, ok = c.getter.(func() (string, error)); !ok {
			c.log.FATAL.Fatalf("invalid type: %T", c.getter)
		}
		g = StringGetter(g)
	}

	return StringGetter(func() (string, error) {
		if c.clock.Since(c.updated) > c.cache {

			val, err := g()
			if err != nil {
				return val, err
			}

			c.updated = c.clock.Now()
			c.val = val
		}

		return c.val.(string), nil
	})
}

// BoolGetter gets bool value
func (c *Cached) BoolGetter() BoolGetter {
	g, ok := c.getter.(BoolGetter)
	if !ok {
		if g, ok = c.getter.(func() (bool, error)); !ok {
			c.log.FATAL.Fatalf("invalid type: %T", g)
		}
		g = BoolGetter(g)
	}

	return BoolGetter(func() (bool, error) {
		if c.clock.Since(c.updated) > c.cache {

			val, err := g()
			if err != nil {
				return val, err
			}

			c.updated = c.clock.Now()
			c.val = val
		}

		return c.val.(bool), nil
	})
}

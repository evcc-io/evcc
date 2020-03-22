package provider

import (
	"time"
)

// CacheGetter wraps a getter with a cache
type CacheGetter struct {
	updated time.Time
	cache   time.Duration
	getter  interface{}
	val     interface{}
}

// NewCacheGetter wraps a getter with a cache
func NewCacheGetter(getter interface{}, cache time.Duration) *CacheGetter {
	return &CacheGetter{
		getter: getter,
		cache:  cache,
	}
}

// FloatGetter gets float value
func (c *CacheGetter) FloatGetter() (float64, error) {
	if time.Since(c.updated) > c.cache {
		g, ok := c.getter.(FloatGetter)
		if !ok {
			log.FATAL.Fatal("invalid type")
		}

		val, err := g()
		if err != nil {
			return val, err
		}

		c.updated = time.Now()
		c.val = val
	}

	return c.val.(float64), nil
}

// IntGetter gets int value
func (c *CacheGetter) IntGetter() (int64, error) {
	if time.Since(c.updated) > c.cache {
		g, ok := c.getter.(IntGetter)
		if !ok {
			log.FATAL.Fatal("invalid type")
		}

		val, err := g()
		if err != nil {
			return val, err
		}

		c.updated = time.Now()
		c.val = val
	}

	return c.val.(int64), nil
}

// StringGetter gets string value
func (c *CacheGetter) StringGetter() (string, error) {
	if time.Since(c.updated) > c.cache {
		g, ok := c.getter.(StringGetter)
		if !ok {
			log.FATAL.Fatal("invalid type")
		}

		val, err := g()
		if err != nil {
			return val, err
		}

		c.updated = time.Now()
		c.val = val
	}

	return c.val.(string), nil
}

// BoolGetter gets bool value
func (c *CacheGetter) BoolGetter() (bool, error) {
	if time.Since(c.updated) > c.cache {
		g, ok := c.getter.(BoolGetter)
		if !ok {
			log.FATAL.Fatal("invalid type")
		}

		val, err := g()
		if err != nil {
			return val, err
		}

		c.updated = time.Now()
		c.val = val
	}

	return c.val.(bool), nil
}

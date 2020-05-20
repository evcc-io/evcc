package util

import "sync"

// Cache is a data store
type Cache struct {
	sync.Mutex
	val map[string]interface{}
}

// NewCache creates cache
func NewCache() *Cache {
	return &Cache{val: make(map[string]interface{})}
}

// Run adds input channel's values to cache
func (c *Cache) Run(in <-chan Param) {
	for p := range in {
		c.Add(p.Key, p)
	}
}

// All provides a copy of the cached values
func (c *Cache) All() []Param {
	c.Lock()
	defer c.Unlock()

	copy := make([]Param, 0, len(c.val))
	for _, v := range c.val {
		if param, ok := v.(Param); ok {
			copy = append(copy, param)
		}
	}

	return copy
}

// Add entry to cache
func (c *Cache) Add(key string, param Param) {
	c.Lock()
	defer c.Unlock()

	c.val[key] = param
}

// Get entry from cache
func (c *Cache) Get(key string) Param {
	c.Lock()
	defer c.Unlock()

	if val, ok := c.val[key]; ok {
		if param, ok := val.(Param); ok {
			return param
		}
	}

	return Param{}
}

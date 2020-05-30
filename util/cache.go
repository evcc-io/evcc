package util

import (
	"sync"
)

// Cache is a data store
type Cache struct {
	sync.Mutex
	val map[string]Param
}

type CacheState struct {
	Site       map[string]interface{}   `json:"site"`
	LoadPoints []map[string]interface{} `json:"loadPoints"`
}

// NewCache creates cache
func NewCache() *Cache {
	return &Cache{val: make(map[string]Param)}
}

// Run adds input channel's values to cache
func (c *Cache) Run(in <-chan Param) {
	for p := range in {
		c.Add(p.Key, p)
	}
}

// State provides a structured copy of the cached values
func (c *Cache) State() CacheState {
	c.Lock()
	defer c.Unlock()

	cs := CacheState{
		Site:       make(map[string]interface{}),
		LoadPoints: make([]map[string]interface{}, 0),
	}

	lps := make(map[string]map[string]interface{})

	for _, param := range c.val {
		if param.LoadPoint == "" {
			cs.Site[param.Key] = param.Val
		} else {
			lp, ok := lps[param.LoadPoint]
			if !ok {
				lp = make(map[string]interface{})
				lps[param.LoadPoint] = lp
			}
			lp[param.Key] = param.Val
		}
	}

	for name, lp := range lps {
		lp["name"] = name
		cs.LoadPoints = append(cs.LoadPoints, lp)
	}

	return cs
}

// All provides a copy of the cached values
func (c *Cache) All() []Param {
	c.Lock()
	defer c.Unlock()

	copy := make([]Param, 0, len(c.val))
	for _, val := range c.val {
		copy = append(copy, val)
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
		return val
	}

	return Param{}
}

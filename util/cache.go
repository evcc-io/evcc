package util

import (
	"fmt"
	"sync"
)

// Cache is a data store
type Cache struct {
	sync.Mutex
	val map[string]Param
}

// NewCache creates cache
func NewCache() *Cache {
	return &Cache{
		val: make(map[string]Param),
	}
}

// Run adds input channel's values to cache
func (c *Cache) Run(in <-chan Param) {
	log := NewLogger("cache")

	for p := range in {
		key := p.Key
		if p.LoadPoint != nil {
			key = fmt.Sprintf("lp-%d/%s", *p.LoadPoint+1, key)
		} else if len(p.Meter) > 0 {
			key = fmt.Sprintf("m-%s/%s", p.Meter, key)
		}
		log.TRACE.Printf("%s: %v", key, p.Val)
		c.Add(p.UniqueID(), p)
	}
}

// State provides a structured copy of the cached values
// Loadpoints are aggregated as loadpoints array
// Meters are aggregated as meters array
func (c *Cache) State() map[string]interface{} {
	c.Lock()
	defer c.Unlock()

	res := map[string]interface{}{}
	lps := make(map[int]map[string]interface{})
	ms := make(map[string]map[string]interface{})

	for _, param := range c.val {
		if param.LoadPoint != nil {
			lp, ok := lps[*param.LoadPoint]
			if !ok {
				lp = make(map[string]interface{})
				lps[*param.LoadPoint] = lp
			}
			lp[param.Key] = param.Val
		} else if len(param.Meter) > 0 {
			m, ok := ms[param.Meter]
			if !ok {
				m = make(map[string]interface{})
				ms[param.Meter] = m
			}
			m[param.Key] = param.Val
		} else {
			res[param.Key] = param.Val
		}
	}

	// convert map to array
	loadpoints := make([]map[string]interface{}, len(lps))
	for id, lp := range lps {
		loadpoints[id] = lp
	}
	res["loadpoints"] = loadpoints

	// convert map to array
	meters := make([]map[string]interface{}, len(ms))
	i := 0
	for _, m := range ms {
		meters[i] = m
		i++
	}
	res["meters"] = meters

	return res
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

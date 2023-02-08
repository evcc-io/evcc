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

// flush is the value type used as parameter for flushing the cache.
// Flushing is implemented by closing the channel. At this time, it is guaranteed
// that the cache has catched up processing all pending messages.
type flush chan struct{}

// Flusher returns a new flush channel
func Flusher() flush {
	return make(flush)
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
		if flushC, ok := p.Val.(flush); ok {
			close(flushC)
			continue
		}

		key := p.Key
		if p.Loadpoint != nil {
			key = fmt.Sprintf("lp-%d/%s", *p.Loadpoint+1, key)
		}

		log.TRACE.Printf("%s: %v", key, p.Val)
		c.Add(p.UniqueID(), p)
	}
}

// State provides a structured copy of the cached values
// Loadpoints are aggregated as loadpoints array
func (c *Cache) State() map[string]interface{} {
	c.Lock()
	defer c.Unlock()

	res := map[string]interface{}{}
	lps := make(map[int]map[string]interface{})

	for _, param := range c.val {
		if param.Loadpoint == nil {
			res[param.Key] = param.Val
		} else {
			lp, ok := lps[*param.Loadpoint]
			if !ok {
				lp = make(map[string]interface{})
				lps[*param.Loadpoint] = lp
			}
			lp[param.Key] = param.Val
		}
	}

	// convert map to array
	loadpoints := make([]map[string]interface{}, len(lps))
	for id, lp := range lps {
		loadpoints[id] = lp
	}
	res["loadpoints"] = loadpoints

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

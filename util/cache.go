package util

import (
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/util/encode"
	"golang.org/x/exp/maps"
)

// Cache is a data store
type Cache struct {
	mu  sync.RWMutex
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

// State provides a structured copy of the cached values.
// Loadpoints are aggregated as loadpoints array.
// Result values are formatted using encoder.
func (c *Cache) State(enc encode.Encoder) map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := make(map[string]any)
	lps := make(map[int]map[string]any)

	for _, param := range c.val {
		if param.Loadpoint == nil {
			res[param.Key] = enc.Encode(param.Val)
		} else {
			lp, ok := lps[*param.Loadpoint]
			if !ok {
				lp = make(map[string]any)
				lps[*param.Loadpoint] = lp
			}
			lp[param.Key] = enc.Encode(param.Val)
		}
	}

	// convert map to array
	loadpoints := make([]map[string]any, len(lps))
	for id, lp := range lps {
		loadpoints[id] = lp
	}
	res["loadpoints"] = loadpoints

	return res
}

// All provides a copy of the cached values
func (c *Cache) All() []Param {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return maps.Values(c.val)
}

// Add entry to cache
func (c *Cache) Add(key string, param Param) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.val[key] = param
}

// Get entry from cache
func (c *Cache) Get(key string) Param {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if val, ok := c.val[key]; ok {
		return val
	}

	return Param{}
}

package util

import (
	"maps"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/evcc-io/evcc/util/encode"
)

// Param is the broadcast channel data type
type Param struct {
	Loadpoint *int
	Key       string
	Val       any
}

// UniqueID returns unique identifier for parameter Loadpoint/Key combination
func (p Param) UniqueID() string {
	if p.Loadpoint != nil {
		return strconv.Itoa(*p.Loadpoint) + "." + p.Key
	}

	return p.Key
}

// ParamCache is a data store
type ParamCache struct {
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
func NewParamCache() *ParamCache {
	return &ParamCache{
		val: make(map[string]Param),
	}
}

// defaultParamCache is the process-wide value cache (the same store that backs
// /api/state), registered once at startup in cmd/root.go.
//
// It is intentionally a package-level singleton rather than a threaded dependency:
// the cache is a genuine process singleton, and its only in-process readers are
// plugins (e.g. the state source) instantiated deep in the template/registry chain
// - passing a reference down would touch every device constructor across all device
// types for no functional gain. Access is lock-free and nil-safe (no value before
// registration, e.g. in CLI subcommands or tests).
var defaultParamCache atomic.Pointer[ParamCache]

// SetDefaultParamCache registers the process-wide value cache. Called once at
// startup; tests use it to set/reset the cache.
func SetDefaultParamCache(c *ParamCache) {
	defaultParamCache.Store(c)
}

// DefaultParamCacheValue returns the cached value for key, or nil if no cache is
// registered or the key is unknown.
func DefaultParamCacheValue(key string) any {
	c := defaultParamCache.Load()
	if c == nil {
		return nil
	}
	return c.Get(key).Val
}

// Run adds input channel's values to cache
func (c *ParamCache) Run(in <-chan Param) {
	for p := range in {
		if flushC, ok := p.Val.(flush); ok {
			close(flushC)
			continue
		}

		c.Add(p.UniqueID(), p)
	}
}

// State provides a structured copy of the cached values.
// Loadpoints are aggregated as loadpoints array.
// Result values are formatted using encoder.
func (c *ParamCache) State(enc encode.Encoder) map[string]any {
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
func (c *ParamCache) All() []Param {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return slices.Collect(maps.Values(c.val))
}

// Add entry to cache
func (c *ParamCache) Add(key string, param Param) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.val[key] = param
}

// Get entry from cache
func (c *ParamCache) Get(key string) Param {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if val, ok := c.val[key]; ok {
		return val
	}

	return Param{}
}

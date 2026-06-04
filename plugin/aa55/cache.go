package aa55

import (
	"sync"
	"time"
)

const cacheTTL = 2 * time.Second

// cache is the package-level response cache shared across all AA55UDP plugin
// instances. Sharing at package level ensures that multiple source blocks for
// the same (host, pdu) pair — e.g. the four Ppv string registers all using
// READ 125 @ 0x891C — share one UDP exchange per poll cycle.
//
// TTL is 2 s: long enough to serve all source blocks within one evcc poll
// cycle (which completes in well under 1 s), short enough that the next cycle
// always fetches fresh data.
var cache = newResponseCache()

type cacheEntry struct {
	payload   []byte
	expiresAt time.Time
}

type responseCache struct {
	mu   sync.Mutex
	data map[string]cacheEntry
}

func newResponseCache() *responseCache {
	return &responseCache{data: make(map[string]cacheEntry)}
}

// get returns the cached payload if it exists and is fresh, or (nil, false)
// otherwise. Expired entries are deleted on access. The map lookup
// m[string(key)] is alloc-free — the Go compiler elides the conversion.
func (c *responseCache) get(key []byte) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.data[string(key)]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(c.data, string(key))
		return nil, false
	}
	return entry.payload, true
}

// put inserts or overwrites a payload in the cache.
func (c *responseCache) put(key, payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[string(key)] = cacheEntry{
		payload:   payload,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

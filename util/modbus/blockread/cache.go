package blockread

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// DefaultTTL serves all sources within one poll cycle (which completes well
// under 1s) while forcing fresh data on the next cycle.
const DefaultTTL = 2 * time.Second

// Cache is a TTL response cache with single-flight de-duplication. Sources
// covering the same key share one device exchange per poll cycle.
type Cache struct {
	ttl    time.Duration
	mu     sync.Mutex
	data   map[string]entry
	flight singleflight.Group
}

type entry struct {
	payload   []byte
	expiresAt time.Time
}

// NewCache returns a Cache that holds entries for ttl.
func NewCache(ttl time.Duration) *Cache {
	return &Cache{ttl: ttl, data: make(map[string]entry)}
}

// Fetch returns the cached payload for key if it is fresh. On a miss, load is
// invoked exactly once across all concurrent callers sharing the same key.
func (c *Cache) Fetch(key string, load func() ([]byte, error)) ([]byte, bool, error) {
	if payload, ok := c.get(key); ok {
		return payload, true, nil
	}

	payload, err, _ := c.flight.Do(key, func() (any, error) {
		// re-check under the flight: a prior flight may have populated the
		// cache between our miss above and acquiring the call.
		if payload, ok := c.get(key); ok {
			return payload, nil
		}
		payload, err := load()
		if err != nil {
			return nil, err
		}
		c.put(key, payload)
		return payload, nil
	})
	if err != nil {
		return nil, false, err
	}
	return payload.([]byte), false, nil
}

// get returns the cached payload if it exists and is fresh. Expired entries are
// deleted on access.
func (c *Cache) get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.data[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(c.data, key)
		return nil, false
	}
	return entry.payload, true
}

// put inserts or overwrites a payload in the cache.
func (c *Cache) put(key string, payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = entry{
		payload:   payload,
		expiresAt: time.Now().Add(c.ttl),
	}
}

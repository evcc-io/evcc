package modbus

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Cache is a TTL response cache with single-flight de-duplication, sharing one
// device exchange per key. The TTL is caller-chosen (depends on poll cadence).
type Cache struct {
	ttl    time.Duration
	mu     sync.Mutex
	data   map[string]cacheEntry
	flight singleflight.Group
}

type cacheEntry struct {
	payload   []byte
	expiresAt time.Time
}

// NewCache returns a Cache that holds entries for ttl.
func NewCache(ttl time.Duration) *Cache {
	return &Cache{ttl: ttl, data: make(map[string]cacheEntry)}
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

	e, ok := c.data[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expiresAt) {
		delete(c.data, key)
		return nil, false
	}
	return e.payload, true
}

// Clear drops all cached entries. Callers use this after a write to force the
// next read to fetch fresh values instead of serving a stale cached payload.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	clear(c.data)
}

// put inserts or overwrites a payload in the cache.
func (c *Cache) put(key string, payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheEntry{
		payload:   payload,
		expiresAt: time.Now().Add(c.ttl),
	}
}

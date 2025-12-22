package oauth

import (
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"
)

// TokenSourceCache provides thread-safe caching of oauth2.TokenSource instances
// keyed by username. This allows multiple components using the same username
// to share a single TokenSource, avoiding duplicate authentication.
type TokenSourceCache struct {
	mu    sync.Mutex
	cache map[string]oauth2.TokenSource
	group singleflight.Group
}

// NewTokenSourceCache creates a new TokenSourceCache instance.
func NewTokenSourceCache() *TokenSourceCache {
	return &TokenSourceCache{
		cache: make(map[string]oauth2.TokenSource),
	}
}

// GetOrCreate atomically gets or creates a TokenSource for the given user.
// If multiple goroutines call this concurrently for the same user, only one
// will execute createFn and others will wait for and share the result.
// This prevents duplicate authentication requests when multiple chargers
// are initialized concurrently with the same credentials.
func (c *TokenSourceCache) GetOrCreate(user string, createFn func() (oauth2.TokenSource, error)) (oauth2.TokenSource, error) {
	result, err, _ := c.group.Do(user, func() (interface{}, error) {
		// Check cache first to avoid duplicate work
		c.mu.Lock()
		if ts := c.cache[user]; ts != nil {
			c.mu.Unlock()
			return ts, nil
		}
		c.mu.Unlock()

		// Create new token source
		ts, err := createFn()
		if err != nil {
			return nil, err
		}

		// Store in cache
		c.mu.Lock()
		c.cache[user] = ts
		c.mu.Unlock()

		return ts, nil
	})

	if err != nil {
		return nil, err
	}
	return result.(oauth2.TokenSource), nil
}

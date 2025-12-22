package oauth

import (
	"sync"

	"golang.org/x/oauth2"
)

// TokenSourceCache provides thread-safe caching of oauth2.TokenSource instances
// keyed by username. This allows multiple components using the same username
// to share a single TokenSource, avoiding duplicate authentication.
type TokenSourceCache struct {
	mu    sync.RWMutex
	cache map[string]oauth2.TokenSource
}

// NewTokenSourceCache creates a new TokenSourceCache instance.
func NewTokenSourceCache() *TokenSourceCache {
	return &TokenSourceCache{
		cache: make(map[string]oauth2.TokenSource),
	}
}

// GetOrCreate atomically gets or creates a TokenSource for the given user.
// If multiple goroutines call this concurrently for the same user, the first
// one will execute createFn and others will wait for and share the result.
// This prevents duplicate authentication requests when multiple chargers
// are initialized concurrently with the same credentials.
func (c *TokenSourceCache) GetOrCreate(user string, createFn func() (oauth2.TokenSource, error)) (oauth2.TokenSource, error) {
	c.mu.RLock()
	ts := c.cache[user]
	c.mu.RUnlock()

	if ts != nil {
		return ts, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check: another goroutine might have created it while we waited for the lock
	if ts := c.cache[user]; ts != nil {
		return ts, nil
	}

	ts, err := createFn()
	if err != nil {
		return nil, err
	}

	c.cache[user] = ts
	return ts, nil
}

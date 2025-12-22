package oauth

import (
	"sync"

	"golang.org/x/oauth2"
)

// TokenSourceCache provides thread-safe caching of oauth2.TokenSource instances
// keyed by username. This allows multiple components using the same username
// to share a single TokenSource, avoiding duplicate authentication.
type TokenSourceCache struct {
	mu    sync.Mutex
	cache map[string]oauth2.TokenSource
}

// NewTokenSourceCache creates a new TokenSourceCache instance.
func NewTokenSourceCache() *TokenSourceCache {
	return &TokenSourceCache{
		cache: make(map[string]oauth2.TokenSource),
	}
}

// Get retrieves a cached TokenSource for the given user.
// Returns nil if no TokenSource is found for the given key.
func (c *TokenSourceCache) Get(user string) oauth2.TokenSource {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.cache[user]
}

// Set stores a TokenSource for the given user in the cache.
func (c *TokenSourceCache) Set(user string, ts oauth2.TokenSource) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[user] = ts
}

// Clear removes the cached TokenSource for the given user.
// This should be called when credentials change or when reconfiguring.
func (c *TokenSourceCache) Clear(user string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, user)
}

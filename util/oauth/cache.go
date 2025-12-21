package oauth

import (
	"sync"

	"golang.org/x/oauth2"
)

// TokenSourceCache provides thread-safe caching of oauth2.TokenSource instances
// keyed by user credentials. This allows multiple components using the same
// credentials to share a single TokenSource, avoiding duplicate authentication.
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

// Get retrieves a cached TokenSource for the given credentials.
// Returns the TokenSource and true if found, nil and false otherwise.
func (c *TokenSourceCache) Get(user, password string) (oauth2.TokenSource, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := CredentialsCacheKey(user, password)
	ts, exists := c.cache[key]
	return ts, exists
}

// Set stores a TokenSource for the given credentials in the cache.
func (c *TokenSourceCache) Set(user, password string, ts oauth2.TokenSource) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := CredentialsCacheKey(user, password)
	c.cache[key] = ts
}

// Clear removes the cached TokenSource for the given credentials.
// This should be called when credentials change or when reconfiguring.
func (c *TokenSourceCache) Clear(user, password string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := CredentialsCacheKey(user, password)
	delete(c.cache, key)
}

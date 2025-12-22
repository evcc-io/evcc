package oauth

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

// stubTokenSource is a test implementation of oauth2.TokenSource
type stubTokenSource struct{}

func (s *stubTokenSource) Token() (*oauth2.Token, error) {
	return nil, nil
}

func TestTokenSourceCache_Clear(t *testing.T) {
	cache := NewTokenSourceCache()
	user := "test@example.com"

	// Manually add an entry to the cache
	ts := &stubTokenSource{}
	cache.Set(user, ts)

	// Verify entry exists
	assert.NotNil(t, cache.Get(user), "cache entry should exist before clearing")

	// Clear the cache
	cache.Clear(user)

	// Verify entry is removed
	assert.Nil(t, cache.Get(user), "cache entry should be removed after clearing")
}

func TestTokenSourceCache_Clear_NotExisting(t *testing.T) {
	cache := NewTokenSourceCache()

	// Clearing non-existent entry should not panic
	assert.NotPanics(t, func() {
		cache.Clear("nonexistent@example.com")
	}, "clearing non-existent cache entry should not panic")
}

func TestTokenSourceCache_Clear_DifferentUsers(t *testing.T) {
	cache := NewTokenSourceCache()
	user1 := "user1@example.com"
	user2 := "user2@example.com"

	// Add two different entries
	cache.Set(user1, &stubTokenSource{})
	cache.Set(user2, &stubTokenSource{})

	// Clear only the first entry
	cache.Clear(user1)

	// Verify only the first entry is removed
	assert.Nil(t, cache.Get(user1), "first cache entry should be removed")
	assert.NotNil(t, cache.Get(user2), "second cache entry should still exist")
}

func TestTokenSourceCache_GetSet(t *testing.T) {
	cache := NewTokenSourceCache()
	user := "test@example.com"

	// Initially, cache should be empty
	value := cache.Get(user)
	assert.Nil(t, value, "cache should be empty initially")

	// Set a non-nil TokenSource
	expected := &stubTokenSource{}
	cache.Set(user, expected)

	// Get should return the same TokenSource instance
	actual := cache.Get(user)
	assert.NotNil(t, actual, "cache should contain the entry after Set")
	assert.Same(t, expected, actual, "cached TokenSource instance should be the same as the one stored")
}

func TestTokenSourceCache_ConcurrentGetSet(t *testing.T) {
	cache := NewTokenSourceCache()
	user := "concurrent@example.com"

	var wg sync.WaitGroup
	goroutines := 10
	iterations := 100

	// Launch multiple goroutines that concurrently Get and Set
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ts := &stubTokenSource{}
			for j := 0; j < iterations; j++ {
				cache.Set(user, ts)
				assert.NotNil(t, cache.Get(user), "cache should contain entry during concurrent access")
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is in a consistent state
	assert.NotNil(t, cache.Get(user), "cache should contain entry after concurrent operations")
}

func TestTokenSourceCache_ConcurrentClearWithReadWrite(t *testing.T) {
	cache := NewTokenSourceCache()
	user := "cleartest@example.com"

	// Pre-populate the cache
	cache.Set(user, &stubTokenSource{})

	var wg sync.WaitGroup
	goroutines := 10
	iterations := 50

	// Launch goroutines that read/write
	for i := 0; i < goroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ts := &stubTokenSource{}
			for j := 0; j < iterations; j++ {
				cache.Set(user, ts)
				cache.Get(user)
			}
		}()
	}

	// Launch goroutines that clear
	for i := 0; i < goroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				cache.Clear(user)
			}
		}()
	}

	// Should not panic or deadlock
	wg.Wait()
}

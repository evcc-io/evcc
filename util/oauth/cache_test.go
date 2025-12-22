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

	// Create an entry using GetOrCreate
	callCount := 0
	ts1 := &stubTokenSource{}
	result, err := cache.GetOrCreate(user, func() (oauth2.TokenSource, error) {
		callCount++
		return ts1, nil
	})
	assert.NoError(t, err)
	assert.Same(t, ts1, result, "GetOrCreate should return the created TokenSource")
	assert.Equal(t, 1, callCount, "createFn should be called once")

	// Clear the cache
	cache.Clear(user)

	// Verify entry is removed by checking that createFn is called again
	ts2 := &stubTokenSource{}
	result, err = cache.GetOrCreate(user, func() (oauth2.TokenSource, error) {
		callCount++
		return ts2, nil
	})
	assert.NoError(t, err)
	assert.Same(t, ts2, result, "GetOrCreate should create a new TokenSource after clearing")
	assert.Equal(t, 2, callCount, "createFn should be called again after clearing")
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

	// Create two different entries
	ts1 := &stubTokenSource{}
	ts2 := &stubTokenSource{}
	_, err := cache.GetOrCreate(user1, func() (oauth2.TokenSource, error) {
		return ts1, nil
	})
	assert.NoError(t, err)
	result2, err := cache.GetOrCreate(user2, func() (oauth2.TokenSource, error) {
		return ts2, nil
	})
	assert.NoError(t, err)

	// Clear only the first entry
	cache.Clear(user1)

	// Verify only the first entry is removed by creating a new one
	newTs1 := &stubTokenSource{}
	result1, err := cache.GetOrCreate(user1, func() (oauth2.TokenSource, error) {
		return newTs1, nil
	})
	assert.NoError(t, err)
	assert.Same(t, newTs1, result1, "GetOrCreate should create a new TokenSource for user1 after clearing")

	// Verify second entry still exists (GetOrCreate should return cached value)
	cachedTs2, err := cache.GetOrCreate(user2, func() (oauth2.TokenSource, error) {
		return &stubTokenSource{}, nil // This shouldn't be called
	})
	assert.NoError(t, err)
	assert.Same(t, result2, cachedTs2, "second cache entry should still exist and return cached value")
}

func TestTokenSourceCache_GetOrCreate(t *testing.T) {
	cache := NewTokenSourceCache()
	user := "test@example.com"

	// First call should create the TokenSource
	expected := &stubTokenSource{}
	callCount := 0
	result, err := cache.GetOrCreate(user, func() (oauth2.TokenSource, error) {
		callCount++
		return expected, nil
	})
	assert.NoError(t, err)
	assert.Same(t, expected, result, "GetOrCreate should return the created TokenSource")
	assert.Equal(t, 1, callCount, "createFn should be called once")

	// Second call should return cached value without calling createFn
	result2, err := cache.GetOrCreate(user, func() (oauth2.TokenSource, error) {
		callCount++
		return &stubTokenSource{}, nil // This shouldn't be called
	})
	assert.NoError(t, err)
	assert.Same(t, expected, result2, "GetOrCreate should return the cached TokenSource")
	assert.Equal(t, 1, callCount, "createFn should not be called again for cached entry")
}

func TestTokenSourceCache_ConcurrentGetOrCreate(t *testing.T) {
	cache := NewTokenSourceCache()
	user := "concurrent@example.com"

	var wg sync.WaitGroup
	var callCount sync.Map
	goroutines := 10

	expected := &stubTokenSource{}

	// Launch multiple goroutines that concurrently call GetOrCreate
	// The singleflight mechanism should ensure createFn is only called once
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			result, err := cache.GetOrCreate(user, func() (oauth2.TokenSource, error) {
				// Track that this goroutine's createFn was called
				callCount.Store(id, true)
				return expected, nil
			})
			assert.NoError(t, err)
			assert.Same(t, expected, result, "all goroutines should get the same TokenSource instance")
		}(i)
	}

	wg.Wait()

	// Count how many goroutines had their createFn called
	count := 0
	callCount.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	// Only one goroutine should have had its createFn called due to singleflight
	assert.Equal(t, 1, count, "createFn should only be called once even with concurrent GetOrCreate calls")
}

func TestTokenSourceCache_ConcurrentClearWithGetOrCreate(t *testing.T) {
	cache := NewTokenSourceCache()
	user := "cleartest@example.com"

	// Pre-populate the cache
	_, err := cache.GetOrCreate(user, func() (oauth2.TokenSource, error) {
		return &stubTokenSource{}, nil
	})
	assert.NoError(t, err)

	var wg sync.WaitGroup
	goroutines := 10
	iterations := 50

	// Launch goroutines that call GetOrCreate
	for i := 0; i < goroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, err := cache.GetOrCreate(user, func() (oauth2.TokenSource, error) {
					return &stubTokenSource{}, nil
				})
				assert.NoError(t, err)
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

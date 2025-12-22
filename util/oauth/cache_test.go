package oauth

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

// stubTokenSource is a test implementation of oauth2.TokenSource
type stubTokenSource struct{}

func (s *stubTokenSource) Token() (*oauth2.Token, error) {
	return nil, nil
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

	// Starting gate to ensure goroutines start concurrently
	start := make(chan struct{})

	// Launch multiple goroutines that concurrently call GetOrCreate
	// The singleflight mechanism should ensure createFn is only called once
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start // Wait for starting signal
			result, err := cache.GetOrCreate(user, func() (oauth2.TokenSource, error) {
				// Track that this goroutine's createFn was called
				callCount.Store(id, true)
				// Simulate realistic OAuth token creation which involves network calls
				time.Sleep(10 * time.Millisecond)
				return expected, nil
			})
			assert.NoError(t, err)
			assert.Same(t, expected, result, "all goroutines should get the same TokenSource instance")
		}(i)
	}

	close(start) // Release all goroutines at once
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

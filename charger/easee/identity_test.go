package easee

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestCacheKey(t *testing.T) {
	// Test that same credentials produce same key
	key1 := cacheKey("user@example.com", "password123")
	key2 := cacheKey("user@example.com", "password123")
	assert.Equal(t, key1, key2, "same credentials should produce same cache key")

	// Test that different passwords produce different keys
	key3 := cacheKey("user@example.com", "different_password")
	assert.NotEqual(t, key1, key3, "different passwords should produce different cache keys")

	// Test that different users produce different keys
	key4 := cacheKey("another@example.com", "password123")
	assert.NotEqual(t, key1, key4, "different users should produce different cache keys")

	// Test that key is deterministic
	key5 := cacheKey("user@example.com", "password123")
	assert.Equal(t, key1, key5, "cache key should be deterministic")

	// Test that key is not empty
	assert.NotEmpty(t, key1, "cache key should not be empty")

	// Test that key is hexadecimal SHA256 (64 characters)
	assert.Len(t, key1, 64, "SHA256 hash should be 64 hex characters")
}

func TestClearTokenCache(t *testing.T) {
	// Reset cache before test
	tokenSourceMu.Lock()
	oldCache := tokenSourceCache
	tokenSourceCache = make(map[string]oauth2.TokenSource)
	tokenSourceMu.Unlock()
	defer func() {
		tokenSourceMu.Lock()
		tokenSourceCache = oldCache
		tokenSourceMu.Unlock()
	}()

	user := "test@example.com"
	password := "testpass"

	// Manually add an entry to the cache
	key := cacheKey(user, password)
	tokenSourceMu.Lock()
	tokenSourceCache[key] = nil // dummy value
	tokenSourceMu.Unlock()

	// Verify entry exists
	tokenSourceMu.Lock()
	_, exists := tokenSourceCache[key]
	tokenSourceMu.Unlock()
	assert.True(t, exists, "cache entry should exist before clearing")

	// Clear the cache
	ClearTokenCache(user, password)

	// Verify entry is removed
	tokenSourceMu.Lock()
	_, exists = tokenSourceCache[key]
	tokenSourceMu.Unlock()
	assert.False(t, exists, "cache entry should be removed after clearing")
}

func TestClearTokenCache_NotExisting(t *testing.T) {
	// Reset cache before test
	tokenSourceMu.Lock()
	oldCache := tokenSourceCache
	tokenSourceCache = make(map[string]oauth2.TokenSource)
	tokenSourceMu.Unlock()
	defer func() {
		tokenSourceMu.Lock()
		tokenSourceCache = oldCache
		tokenSourceMu.Unlock()
	}()

	// Clearing non-existent entry should not panic
	assert.NotPanics(t, func() {
		ClearTokenCache("nonexistent@example.com", "password")
	}, "clearing non-existent cache entry should not panic")
}

func TestClearTokenCache_DifferentCredentials(t *testing.T) {
	// Reset cache before test
	tokenSourceMu.Lock()
	oldCache := tokenSourceCache
	tokenSourceCache = make(map[string]oauth2.TokenSource)
	tokenSourceMu.Unlock()
	defer func() {
		tokenSourceMu.Lock()
		tokenSourceCache = oldCache
		tokenSourceMu.Unlock()
	}()

	user1 := "user1@example.com"
	password1 := "pass1"
	user2 := "user2@example.com"
	password2 := "pass2"

	// Add two different entries
	key1 := cacheKey(user1, password1)
	key2 := cacheKey(user2, password2)
	tokenSourceMu.Lock()
	tokenSourceCache[key1] = nil
	tokenSourceCache[key2] = nil
	tokenSourceMu.Unlock()

	// Clear only the first entry
	ClearTokenCache(user1, password1)

	// Verify only the first entry is removed
	tokenSourceMu.Lock()
	_, exists1 := tokenSourceCache[key1]
	_, exists2 := tokenSourceCache[key2]
	tokenSourceMu.Unlock()

	assert.False(t, exists1, "first cache entry should be removed")
	assert.True(t, exists2, "second cache entry should still exist")
}

func TestClearTokenCache_PasswordChange(t *testing.T) {
	// Reset cache before test
	tokenSourceMu.Lock()
	oldCache := tokenSourceCache
	tokenSourceCache = make(map[string]oauth2.TokenSource)
	tokenSourceMu.Unlock()
	defer func() {
		tokenSourceMu.Lock()
		tokenSourceCache = oldCache
		tokenSourceMu.Unlock()
	}()

	user := "user@example.com"
	oldPassword := "oldpass"
	newPassword := "newpass"

	// Add entry with old password
	oldKey := cacheKey(user, oldPassword)
	tokenSourceMu.Lock()
	tokenSourceCache[oldKey] = nil
	tokenSourceMu.Unlock()

	// Clear cache with new password should not affect old entry
	ClearTokenCache(user, newPassword)

	tokenSourceMu.Lock()
	_, oldExists := tokenSourceCache[oldKey]
	tokenSourceMu.Unlock()

	assert.True(t, oldExists, "old cache entry should still exist when clearing with different password")

	// Clear with correct old password should remove it
	ClearTokenCache(user, oldPassword)

	tokenSourceMu.Lock()
	_, oldExists = tokenSourceCache[oldKey]
	tokenSourceMu.Unlock()

	assert.False(t, oldExists, "old cache entry should be removed when clearing with correct password")
}

func TestCacheKeyConcurrency(t *testing.T) {
	// Test that cacheKey is safe for concurrent use
	user := "concurrent@example.com"
	password := "concurrentpass"

	var wg sync.WaitGroup
	results := make([]string, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = cacheKey(user, password)
		}(i)
	}

	wg.Wait()

	// All results should be identical
	expected := results[0]
	for i, result := range results {
		assert.Equal(t, expected, result, "result at index %d should match", i)
	}
}

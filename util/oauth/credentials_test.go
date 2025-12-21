package oauth

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCredentialsCacheKey(t *testing.T) {
	// Test that same credentials produce same key
	key1 := CredentialsCacheKey("user@example.com", "password123")
	key2 := CredentialsCacheKey("user@example.com", "password123")
	assert.Equal(t, key1, key2, "same credentials should produce same cache key")

	// Test that different passwords produce different keys
	key3 := CredentialsCacheKey("user@example.com", "different_password")
	assert.NotEqual(t, key1, key3, "different passwords should produce different cache keys")

	// Test that different users produce different keys
	key4 := CredentialsCacheKey("another@example.com", "password123")
	assert.NotEqual(t, key1, key4, "different users should produce different cache keys")

	// Test that key is deterministic
	key5 := CredentialsCacheKey("user@example.com", "password123")
	assert.Equal(t, key1, key5, "cache key should be deterministic")

	// Test that key is not empty
	assert.NotEmpty(t, key1, "cache key should not be empty")

	// Test that key is hexadecimal SHA256 (64 characters)
	assert.Len(t, key1, 64, "SHA256 hash should be 64 hex characters")
}

func TestCredentialsCacheKeyConcurrency(t *testing.T) {
	// Test that CredentialsCacheKey is safe for concurrent use
	user := "concurrent@example.com"
	password := "concurrentpass"

	var wg sync.WaitGroup
	results := make([]string, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = CredentialsCacheKey(user, password)
		}(i)
	}

	wg.Wait()

	// All results should be identical
	expected := results[0]
	for i, result := range results {
		assert.Equal(t, expected, result, "result at index %d should match", i)
	}
}

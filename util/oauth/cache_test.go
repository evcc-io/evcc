package oauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenSourceCache_Clear(t *testing.T) {
	cache := NewTokenSourceCache()
	user := "test@example.com"
	password := "testpass"

	// Manually add an entry to the cache
	cache.Set(user, password, nil) // dummy value

	// Verify entry exists
	_, exists := cache.Get(user, password)
	assert.True(t, exists, "cache entry should exist before clearing")

	// Clear the cache
	cache.Clear(user, password)

	// Verify entry is removed
	_, exists = cache.Get(user, password)
	assert.False(t, exists, "cache entry should be removed after clearing")
}

func TestTokenSourceCache_Clear_NotExisting(t *testing.T) {
	cache := NewTokenSourceCache()

	// Clearing non-existent entry should not panic
	assert.NotPanics(t, func() {
		cache.Clear("nonexistent@example.com", "password")
	}, "clearing non-existent cache entry should not panic")
}

func TestTokenSourceCache_Clear_DifferentCredentials(t *testing.T) {
	cache := NewTokenSourceCache()
	user1 := "user1@example.com"
	password1 := "pass1"
	user2 := "user2@example.com"
	password2 := "pass2"

	// Add two different entries
	cache.Set(user1, password1, nil)
	cache.Set(user2, password2, nil)

	// Clear only the first entry
	cache.Clear(user1, password1)

	// Verify only the first entry is removed
	_, exists1 := cache.Get(user1, password1)
	_, exists2 := cache.Get(user2, password2)

	assert.False(t, exists1, "first cache entry should be removed")
	assert.True(t, exists2, "second cache entry should still exist")
}

func TestTokenSourceCache_Clear_PasswordChange(t *testing.T) {
	cache := NewTokenSourceCache()
	user := "user@example.com"
	oldPassword := "oldpass"
	newPassword := "newpass"

	// Add entry with old password
	cache.Set(user, oldPassword, nil)

	// Clear cache with new password should not affect old entry
	cache.Clear(user, newPassword)

	_, oldExists := cache.Get(user, oldPassword)
	assert.True(t, oldExists, "old cache entry should still exist when clearing with different password")

	// Clear with correct old password should remove it
	cache.Clear(user, oldPassword)

	_, oldExists = cache.Get(user, oldPassword)
	assert.False(t, oldExists, "old cache entry should be removed when clearing with correct password")
}

func TestTokenSourceCache_GetSet(t *testing.T) {
	cache := NewTokenSourceCache()
	user := "test@example.com"
	password := "testpass"

	// Initially, cache should be empty
	_, exists := cache.Get(user, password)
	assert.False(t, exists, "cache should be empty initially")

	// Set a value
	cache.Set(user, password, nil)

	// Get should return the value
	_, exists = cache.Get(user, password)
	assert.True(t, exists, "cache should contain the entry after Set")
}

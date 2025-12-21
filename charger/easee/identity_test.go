package easee

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClearTokenCache(t *testing.T) {
	user := "test@example.com"
	password := "testpass"

	// Manually add an entry to the cache
	tokenSourceCache.Set(user, password, nil) // dummy value

	// Verify entry exists
	_, exists := tokenSourceCache.Get(user, password)
	assert.True(t, exists, "cache entry should exist before clearing")

	// Clear the cache
	ClearTokenCache(user, password)

	// Verify entry is removed
	_, exists = tokenSourceCache.Get(user, password)
	assert.False(t, exists, "cache entry should be removed after clearing")
}

func TestClearTokenCache_NotExisting(t *testing.T) {
	// Clearing non-existent entry should not panic
	assert.NotPanics(t, func() {
		ClearTokenCache("nonexistent@example.com", "password")
	}, "clearing non-existent cache entry should not panic")
}

func TestClearTokenCache_DifferentCredentials(t *testing.T) {
	user1 := "user1@example.com"
	password1 := "pass1"
	user2 := "user2@example.com"
	password2 := "pass2"

	// Add two different entries
	tokenSourceCache.Set(user1, password1, nil)
	tokenSourceCache.Set(user2, password2, nil)

	// Clear only the first entry
	ClearTokenCache(user1, password1)

	// Verify only the first entry is removed
	_, exists1 := tokenSourceCache.Get(user1, password1)
	_, exists2 := tokenSourceCache.Get(user2, password2)

	assert.False(t, exists1, "first cache entry should be removed")
	assert.True(t, exists2, "second cache entry should still exist")
}

func TestClearTokenCache_PasswordChange(t *testing.T) {
	user := "user@example.com"
	oldPassword := "oldpass"
	newPassword := "newpass"

	// Add entry with old password
	tokenSourceCache.Set(user, oldPassword, nil)

	// Clear cache with new password should not affect old entry
	ClearTokenCache(user, newPassword)

	_, oldExists := tokenSourceCache.Get(user, oldPassword)
	assert.True(t, oldExists, "old cache entry should still exist when clearing with different password")

	// Clear with correct old password should remove it
	ClearTokenCache(user, oldPassword)

	_, oldExists = tokenSourceCache.Get(user, oldPassword)
	assert.False(t, oldExists, "old cache entry should be removed when clearing with correct password")
}

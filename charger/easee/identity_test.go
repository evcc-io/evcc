package easee

import (
	"testing"

	"github.com/evcc-io/evcc/util/oauth"
	"github.com/stretchr/testify/assert"
)

func TestClearTokenCache(t *testing.T) {
	user := "test@example.com"
	password := "testpass"

	// Manually add an entry to the cache
	key := oauth.CredentialsCacheKey(user, password)
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
	key1 := oauth.CredentialsCacheKey(user1, password1)
	key2 := oauth.CredentialsCacheKey(user2, password2)
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
	user := "user@example.com"
	oldPassword := "oldpass"
	newPassword := "newpass"

	// Add entry with old password
	oldKey := oauth.CredentialsCacheKey(user, oldPassword)
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

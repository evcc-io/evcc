package settings

import (
	"testing"

	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) func() {
	// Use in-memory SQLite database for testing
	err := db.NewInstance("sqlite", ":memory:")
	require.NoError(t, err, "Failed to create test database")
	
	err = Init()
	require.NoError(t, err, "Failed to initialize settings")
	
	// Return cleanup function
	return func() {
		// Reset the database instance
		db.Instance = nil
	}
}

// TestDelete_Integration tests the complete Delete() function including database operations
// 
// This test highlights the bug that was fixed in settings.Delete():
// BUG: slices.Delete(settings, idx, idx) did not delete anything from in-memory cache
// FIX: slices.Delete(settings, idx, idx+1) properly deletes from in-memory cache
//
// Before the fix, this test would fail because:
// - Database deletion worked correctly ✅
// - In-memory deletion was broken ❌  
// - String() would return the deleted value instead of ErrNotFound
func TestDelete_Integration(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	
	key := "test_delete_integration"
	value := "test_value_integration"
	
	// Set a value
	SetString(key, value)
	
	// Verify it exists in both memory and database
	res, err := String(key)
	require.NoError(t, err, "Key should exist after setting")
	assert.Equal(t, value, res, "Should return correct value")
	
	// Verify it exists in the settings slice (in-memory cache)
	found := false
	for _, setting := range settings {
		if setting.Key == key {
			found = true
			break
		}
	}
	assert.True(t, found, "Key should exist in in-memory settings slice before deletion")
	
	// Delete it - this is where the bug was
	err = Delete(key)
	require.NoError(t, err, "Delete should not return an error")
	
	// CRITICAL: This is the test that failed before the fix
	// The bug was that entries remained in memory even after "successful" deletion
	_, err = String(key)
	assert.Equal(t, ErrNotFound, err, "Key should not exist in memory after deletion (this was the bug)")
	
	// Verify it's also gone from the in-memory settings slice
	found = false
	for _, setting := range settings {
		if setting.Key == key {
			found = true
			break
		}
	}
	assert.False(t, found, "Key should not exist in in-memory settings slice after deletion")
}

// TestDelete_MultipleEntries_Integration tests deleting from a list with multiple entries
// This ensures the slice deletion doesn't affect other entries
func TestDelete_MultipleEntries_Integration(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	
	// Set multiple values
	keys := []string{"multi_test_1", "multi_test_2", "multi_test_3"}
	values := []string{"value_1", "value_2", "value_3"}
	
	for i, key := range keys {
		SetString(key, values[i])
	}
	
	// Verify all exist
	for i, key := range keys {
		res, err := String(key)
		require.NoError(t, err, "Key %s should exist", key)
		assert.Equal(t, values[i], res, "Key %s should have correct value", key)
	}
	
	// Delete the middle entry (most likely to expose slice deletion bugs)
	err := Delete(keys[1])
	require.NoError(t, err, "Delete should not error")
	
	// Verify middle entry is gone
	_, err = String(keys[1])
	assert.Equal(t, ErrNotFound, err, "Deleted key should not exist")
	
	// Verify first and last entries still exist and have correct values
	res, err := String(keys[0])
	require.NoError(t, err, "First key should still exist")
	assert.Equal(t, values[0], res, "First key should have correct value")
	
	res, err = String(keys[2])
	require.NoError(t, err, "Last key should still exist")
	assert.Equal(t, values[2], res, "Last key should have correct value")
	
	// Verify the in-memory slice has exactly 2 entries and correct keys
	remainingKeys := make([]string, 0)
	for _, setting := range settings {
		if setting.Key == keys[0] || setting.Key == keys[1] || setting.Key == keys[2] {
			remainingKeys = append(remainingKeys, setting.Key)
		}
	}
	assert.Len(t, remainingKeys, 2, "Should have exactly 2 test keys remaining in memory")
	assert.Contains(t, remainingKeys, keys[0], "First key should remain in memory")
	assert.Contains(t, remainingKeys, keys[2], "Last key should remain in memory")
	assert.NotContains(t, remainingKeys, keys[1], "Deleted key should not remain in memory")
}

// TestDelete_NonExistentKey_Integration tests deleting a key that doesn't exist
func TestDelete_NonExistentKey_Integration(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	
	// Try to delete a key that doesn't exist
	err := Delete("this_key_does_not_exist")
	assert.NoError(t, err, "Deleting non-existent key should not return an error")
}

// TestDelete_PersistenceAcrossRestart_Integration verifies deletion persists after "restart" simulation
func TestDelete_PersistenceAcrossRestart_Integration(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	
	key := "persistence_test"
	value := "persistence_value"
	
	// Set and delete a value
	SetString(key, value)
	
	// Verify it exists
	res, err := String(key)
	require.NoError(t, err)
	assert.Equal(t, value, res)
	
	// Delete it
	err = Delete(key)
	require.NoError(t, err)
	
	// Verify it's gone from memory
	_, err = String(key)
	assert.Equal(t, ErrNotFound, err)
	
	// Simulate application restart by reinitializing settings from database
	// This tests that the deletion was properly persisted to the database
	err = Init()
	require.NoError(t, err, "Re-initialization should not error")
	
	// After "restart", the key should still be gone
	_, err = String(key)
	assert.Equal(t, ErrNotFound, err, "Deleted key should not exist after restart simulation")
}
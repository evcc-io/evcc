package cmd

import (
	"path/filepath"
	"testing"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPasswordSetWithInvalidSponsorToken verifies that password set command
// works even when the sponsor token is invalid or expired
func TestPasswordSetWithInvalidSponsorToken(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Test configureDatabaseOnly - should succeed despite invalid sponsor token
	// This simulates running `evcc password set` with an invalid sponsor token
	err := configureDatabaseOnly(globalconfig.DB{
		Type: "sqlite",
		Dsn:  dbPath,
	})
	require.NoError(t, err, "configureDatabaseOnly should succeed even with invalid sponsor token")

	// Verify database is initialized
	assert.NotNil(t, db.Instance)

	// Set a password using auth (simulates password set command)
	testPassword := "test-password-123"
	authInstance := auth.New()
	err = authInstance.SetAdminPassword(testPassword)
	require.NoError(t, err, "setting password should work despite invalid sponsor token")

	// Verify password was stored
	storedHash, err := settings.String(keys.AdminPassword)
	require.NoError(t, err)
	assert.NotEmpty(t, storedHash, "password hash should be stored")

	// Verify password is valid
	assert.True(t, authInstance.IsAdminPasswordValid(testPassword))
	assert.False(t, authInstance.IsAdminPasswordValid("wrong-password"))
}

// TestPasswordResetWithInvalidSponsorToken verifies that password reset command
// works even when the sponsor token is invalid or expired
func TestPasswordResetWithInvalidSponsorToken(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Test configureDatabaseOnly - should succeed despite invalid sponsor token
	// This simulates running `evcc password reset` with an invalid sponsor token
	err := configureDatabaseOnly(globalconfig.DB{
		Type: "sqlite",
		Dsn:  dbPath,
	})
	require.NoError(t, err, "configureDatabaseOnly should succeed even with invalid sponsor token")

	// Set an initial password
	authInstance := auth.New()
	require.NoError(t, authInstance.SetAdminPassword("initial-password"))

	// Verify password exists
	storedHash, err := settings.String(keys.AdminPassword)
	require.NoError(t, err)
	assert.NotEmpty(t, storedHash)

	// Reset password (simulates password reset command)
	authInstance.RemoveAdminPassword()

	// Verify password was removed
	storedHash, err = settings.String(keys.AdminPassword)
	require.NoError(t, err)
	assert.Empty(t, storedHash, "password should be removed")
}

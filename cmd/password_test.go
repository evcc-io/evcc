package cmd

import (
	"path/filepath"
	"testing"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPasswordWithInvalidSponsorToken verifies that password commands work
// even when an invalid sponsor token is present in the database
func TestPasswordWithInvalidSponsorToken(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Setup database (same as password commands do)
	err := configureDatabase(globalconfig.DB{
		Type: "sqlite",
		Dsn:  dbPath,
	})
	require.NoError(t, err)

	// Store invalid sponsor token
	settings.SetString(keys.SponsorToken, "invalid-token")
	storedToken, err := settings.String(keys.SponsorToken)
	require.NoError(t, err)
	assert.Equal(t, "invalid-token", storedToken)

	// Set password - should work despite invalid sponsor token
	authInstance := auth.New()
	err = authInstance.SetAdminPassword("test-password")
	require.NoError(t, err)

	// Verify password works
	assert.True(t, authInstance.IsAdminPasswordValid("test-password"))

	// Reset password - should work despite invalid sponsor token
	authInstance.RemoveAdminPassword()

	// Verify password was removed
	storedHash, err := settings.String(keys.AdminPassword)
	require.NoError(t, err)
	assert.Empty(t, storedHash)
}

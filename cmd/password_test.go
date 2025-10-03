package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util/auth"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPasswordSetWithInvalidSponsorToken verifies that password management
// works when using configureDatabase directly, even with invalid sponsor tokens.
// This test demonstrates that the fix (using configureDatabaseOnly instead of
// configureEnvironment) allows password set/reset to work properly.
func TestPasswordSetWithInvalidSponsorToken(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	configPath := filepath.Join(tmpDir, "evcc.yaml")

	// Create config with invalid sponsor token
	configContent := `
sponsortoken: invalid-token-that-will-fail-validation
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Reset global config
	conf = globalconfig.All{
		Interval: 10 * time.Second,
		Log:      "info",
		Network: globalconfig.Network{
			Schema: "http",
			Host:   "evcc.local",
			Port:   7070,
		},
		Mqtt: globalconfig.Mqtt{
			Topic: "evcc",
		},
		Database: globalconfig.DB{
			Type: "sqlite",
			Dsn:  dbPath,
		},
	}

	// Set config file location
	viper.SetConfigFile(configPath)

	// Create a mock command with database flag
	cmd := &cobra.Command{}
	cmd.Flags().String(flagIgnoreDatabase, "", "")

	// Load config
	err := loadConfigFile(&conf, true)
	require.NoError(t, err)

	// Setup database only (skipping sponsor validation via configureEnvironment)
	// This is what password_set.go and password_reset.go use
	err = configureDatabaseOnly(conf.Database)
	require.NoError(t, err, "configureDatabaseOnly should succeed with invalid sponsor token")

	// Store the invalid token to verify it's present
	settings.SetString(keys.SponsorToken, "invalid-token-that-will-fail-validation")
	storedToken, err := settings.String(keys.SponsorToken)
	require.NoError(t, err)
	assert.Equal(t, "invalid-token-that-will-fail-validation", storedToken)

	// Set password - should succeed despite invalid sponsor token
	testPassword := "test-password-123"
	authInstance := auth.New()
	err = authInstance.SetAdminPassword(testPassword)
	require.NoError(t, err, "password set should work with invalid sponsor token")

	// Verify password was stored correctly
	storedHash, err := settings.String(keys.AdminPassword)
	require.NoError(t, err)
	assert.NotEmpty(t, storedHash)
	assert.True(t, authInstance.IsAdminPasswordValid(testPassword))
	assert.False(t, authInstance.IsAdminPasswordValid("wrong-password"))
}

// TestPasswordResetWithInvalidSponsorToken verifies that password reset
// works when using configureDatabase directly, even with invalid sponsor tokens
func TestPasswordResetWithInvalidSponsorToken(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Initialize database only (skipping sponsor validation)
	err := configureDatabaseOnly(globalconfig.DB{
		Type: "sqlite",
		Dsn:  dbPath,
	})
	require.NoError(t, err)

	// Store invalid sponsor token
	settings.SetString(keys.SponsorToken, "expired-token-xyz")
	storedToken, err := settings.String(keys.SponsorToken)
	require.NoError(t, err)
	assert.Equal(t, "expired-token-xyz", storedToken)

	// Set initial password
	authInstance := auth.New()
	require.NoError(t, authInstance.SetAdminPassword("initial-password"))

	storedHash, err := settings.String(keys.AdminPassword)
	require.NoError(t, err)
	assert.NotEmpty(t, storedHash)

	// Reset password - should succeed despite invalid sponsor token
	authInstance.RemoveAdminPassword()

	// Verify password was removed
	storedHash, err = settings.String(keys.AdminPassword)
	require.NoError(t, err)
	assert.Empty(t, storedHash)
}

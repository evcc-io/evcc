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

// TestPasswordCommandShouldNotUseConfigureEnvironment demonstrates why password
// commands should use configureDatabase instead of configureEnvironment.
// This test documents that configureEnvironment is unnecessarily slow and complex
// for password management operations.
func TestPasswordCommandShouldNotUseConfigureEnvironment(t *testing.T) {
	t.Skip("This test documents the issue but is skipped to avoid slow test runs")

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	configPath := filepath.Join(tmpDir, "evcc.yaml")

	// Create config with invalid sponsor token
	configContent := `
sponsortoken: invalid-token-that-will-fail-validation
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Change to tmpDir to avoid file system issues with locale
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create i18n directory to prevent panic
	require.NoError(t, os.MkdirAll("i18n", 0o755))

	// Reset global config
	conf = globalconfig.All{
		Interval:     10 * time.Second,
		Log:          "info",
		SponsorToken: "invalid-token-that-will-fail-validation",
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

	// Create a mock command
	cmd := &cobra.Command{}
	cmd.Flags().String(flagIgnoreDatabase, "", "")
	cmd.Flags().String(flagHeaders, "", "")
	cmd.PersistentFlags().String(flagTemplate, "", "")
	cmd.PersistentFlags().String(flagTemplateType, "", "")

	// Load config
	err := loadConfigFile(&conf, true)
	require.NoError(t, err)

	// This is what the OLD code does - it calls configureEnvironment
	// This is slow (~8 seconds) and unnecessary for password management
	start := time.Now()
	_ = configureEnvironment(cmd, &conf)
	elapsed := time.Since(start)

	// configureEnvironment doesn't fail hard, but it's slow
	t.Logf("configureEnvironment took %v (should be fast for password commands)", elapsed)
	assert.Greater(t, elapsed.Seconds(), 5.0, "configureEnvironment is unnecessarily slow")
}

// TestPasswordSetWithDatabaseOnlySetup verifies that password management
// works when using configureDatabase directly, even with invalid sponsor tokens.
// This test passes on BOTH branches because it uses the correct approach.
func TestPasswordSetWithDatabaseOnlySetup(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Initialize database directly (the correct way for password commands)
	err := configureDatabase(globalconfig.DB{
		Type: "sqlite",
		Dsn:  dbPath,
	})
	require.NoError(t, err)

	// Store invalid sponsor token to simulate the scenario
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

// TestPasswordResetWithDatabaseOnlySetup verifies that password reset
// works when using configureDatabase directly, even with invalid sponsor tokens
func TestPasswordResetWithDatabaseOnlySetup(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Initialize database directly
	err := configureDatabase(globalconfig.DB{
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

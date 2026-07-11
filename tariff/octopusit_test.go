package tariff

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOctopusItConfigParse(t *testing.T) {
	validConfig := map[string]any{
		"email":         "test@example.com",
		"password":      "testpassword",
		"accountNumber": "A-12345678",
	}

	tariff, err := buildOctopusItFromConfig(validConfig)
	require.NoError(t, err)
	require.NotNil(t, tariff)

	missingEmailConfig := map[string]any{
		"password":      "testpassword",
		"accountNumber": "A-12345678",
	}
	_, err = buildOctopusItFromConfig(missingEmailConfig)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing email")

	missingPasswordConfig := map[string]any{
		"email":         "test@example.com",
		"accountNumber": "A-12345678",
	}
	_, err = buildOctopusItFromConfig(missingPasswordConfig)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing password")

	missingAccountNumberConfig := map[string]any{
		"email":    "test@example.com",
		"password": "testpassword",
	}
	_, err = buildOctopusItFromConfig(missingAccountNumberConfig)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing account number")
}

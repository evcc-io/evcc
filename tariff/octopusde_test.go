package tariff

import (
	"testing"

	"github.com/evcc-io/evcc/util/test"
	"github.com/stretchr/testify/require"
)

func TestOctopusDeConfigParse(t *testing.T) {
	test.SkipCI(t)

	validConfig := map[string]any{
		"email":         "test@example.com",
		"password":      "testpassword",
		"accountNumber": "A-12345678",
	}

	tariff, err := buildOctopusDeFromConfig(validConfig)
	require.NoError(t, err)
	require.NotNil(t, tariff)
	require.Equal(t, "test@example.com", tariff.email)
	require.Equal(t, "testpassword", tariff.password)
	require.Equal(t, "A-12345678", tariff.accountNumber)

	missingEmailConfig := map[string]any{
		"password":      "testpassword",
		"accountNumber": "A-12345678",
	}
	_, err = buildOctopusDeFromConfig(missingEmailConfig)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing email")

	missingPasswordConfig := map[string]any{
		"email":         "test@example.com",
		"accountNumber": "A-12345678",
	}
	_, err = buildOctopusDeFromConfig(missingPasswordConfig)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing password")

	missingAccountNumberConfig := map[string]any{
		"email":    "test@example.com",
		"password": "testpassword",
	}
	_, err = buildOctopusDeFromConfig(missingAccountNumberConfig)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing account number")
}

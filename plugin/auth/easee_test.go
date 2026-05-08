package auth

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

func TestNewEaseeFromConfig_MissingCredentials(t *testing.T) {
	_, err := newEaseeFromConfig(t.Context(), map[string]any{})
	require.ErrorIs(t, err, api.ErrMissingCredentials)
}

func TestNewEaseeFromConfig_AccountWithoutCredentials(t *testing.T) {
	_, err := newEaseeFromConfig(t.Context(), map[string]any{"account": "easee-main"})
	require.ErrorIs(t, err, api.ErrMissingCredentials)
}

func TestNewEaseeFromConfig_MissingPassword(t *testing.T) {
	_, err := newEaseeFromConfig(t.Context(), map[string]any{"user": "x@example.com"})
	require.ErrorIs(t, err, api.ErrMissingCredentials)
}

func TestNewEaseeFromConfig_MissingUser(t *testing.T) {
	_, err := newEaseeFromConfig(t.Context(), map[string]any{"password": "secret"})
	require.ErrorIs(t, err, api.ErrMissingCredentials)
}

func TestNewEaseeFromConfig_ValidCredentials(t *testing.T) {
	ts, err := newEaseeFromConfig(t.Context(), map[string]any{
		"account":  "easee-main",
		"user":     "x@example.com",
		"password": "secret",
	})
	require.NoError(t, err)
	require.NotNil(t, ts)
}

//go:build integration
// +build integration

package toyota

import (
	"os"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestIdentityLogin(t *testing.T) {
	// Skip if no credentials provided
	user := os.Getenv("TOYOTA_USER")
	password := os.Getenv("TOYOTA_PASSWORD")
	if user == "" || password == "" {
		t.Fatal("TOYOTA_USER or TOYOTA_PASSWORD not set")
	}

	log := util.NewLogger("test")
	identity := NewIdentity(log)

	err := identity.Login(user, password)
	require.NoError(t, err)

	// Verify we got a valid token
	token, err := identity.Token()
	require.NoError(t, err)
	require.NotEmpty(t, token.AccessToken)
	require.NotEmpty(t, token.RefreshToken)
	require.True(t, token.Valid())
}

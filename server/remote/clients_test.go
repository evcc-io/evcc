package remote

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func seedClient(t *testing.T, user, pw string, expires *time.Time) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, settings.SetJson(keys.RemoteClients, []persistedClient{{
		Client: Client{Username: user, ExpiresAt: expires},
		Hash:   string(hash),
	}}))
}

func TestAuthenticateCache(t *testing.T) {
	r := &Remote{}
	seedClient(t, "app", "secret", nil)

	assert.True(t, r.Authenticate("app", "secret"))
	_, cached := authCache.Load(loadClients()[0].Hash)
	assert.True(t, cached, "successful check is cached")

	assert.True(t, r.Authenticate("app", "secret"), "cache hit")
	assert.False(t, r.Authenticate("app", "wrong"), "cache does not accept other password")
	assert.False(t, r.Authenticate("nobody", "secret"))

	// recreated client with new password invalidates the cached entry
	seedClient(t, "app", "other", nil)
	assert.False(t, r.Authenticate("app", "secret"))
	assert.True(t, r.Authenticate("app", "other"))

	// expiry is enforced on cache hits too
	past := time.Now().Add(-time.Minute)
	seedClient(t, "app", "other", &past)
	assert.False(t, r.Authenticate("app", "other"))

	// deleted client is rejected even with a live cache entry
	require.NoError(t, settings.SetJson(keys.RemoteClients, []persistedClient{}))
	assert.False(t, r.Authenticate("app", "other"))
}

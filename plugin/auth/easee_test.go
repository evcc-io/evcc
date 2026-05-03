package auth

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// resetEaseeInstances clears the in-process instance cache between tests.
func resetEaseeInstances() {
	easeeInstancesMu.Lock()
	easeeInstances = make(map[string]oauth2.TokenSource)
	easeeInstancesMu.Unlock()
}

func TestEaseeSubject_Stable(t *testing.T) {
	s1 := easeeSubject("user@example.com", "pass")
	s2 := easeeSubject("user@example.com", "pass")
	assert.Equal(t, s1, s2, "same email and password must produce same subject")
}

func TestEaseeSubject_Unique(t *testing.T) {
	s1 := easeeSubject("user1@example.com", "pass")
	s2 := easeeSubject("user2@example.com", "pass")
	assert.NotEqual(t, s1, s2, "different emails must produce different subjects")
}

func TestEaseeSubject_Prefix(t *testing.T) {
	s := easeeSubject("x@example.com", "pass")
	assert.Equal(t, "easee-", s[:6])
}

// TestNewEaseeTokenSource_Deduplication verifies that two calls with the same
// email return the cached token-source without creating a new one.
func TestNewEaseeTokenSource_Deduplication(t *testing.T) {
	defer resetEaseeInstances()

	// Pre-populate the cache as if a previous successful login had occurred.
	subject := easeeSubject("user@example.com", "pass")
	fakeTS := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "fake-at", RefreshToken: "fake-rt"})
	easeeInstancesMu.Lock()
	easeeInstances[subject] = fakeTS
	easeeInstancesMu.Unlock()

	// Both calls must return the same cached instance without making HTTP requests.
	ts1, err := NewEaseeTokenSource("user@example.com", "pass")
	require.NoError(t, err)
	require.NotNil(t, ts1)

	ts2, err := NewEaseeTokenSource("user@example.com", "pass")
	require.NoError(t, err)

	assert.Equal(t, ts1, ts2, "same user must return the same token-source")
}

// TestNewEaseeTokenSource_DifferentPassword verifies that two calls with the same
// email but different passwords return different token-sources.
func TestNewEaseeTokenSource_DifferentPassword(t *testing.T) {
	defer resetEaseeInstances()

	// Pre-populate both cache entries to avoid real HTTP requests.
	for _, pw := range []string{"pass", "newpass"} {
		subject := easeeSubject("user@example.com", pw)
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "fake-at-" + pw, RefreshToken: "fake-rt"})
		easeeInstancesMu.Lock()
		easeeInstances[subject] = ts
		easeeInstancesMu.Unlock()
	}

	ts1, err := NewEaseeTokenSource("user@example.com", "pass")
	require.NoError(t, err)
	require.NotNil(t, ts1)

	ts2, err := NewEaseeTokenSource("user@example.com", "newpass")
	require.NoError(t, err)

	assert.NotEqual(t, ts1, ts2, "same user must return different token-sources when password has changed")
}

// TestNewEaseeTokenSource_DifferentUsers verifies that different emails result
// in separate cache entries.
func TestNewEaseeTokenSource_DifferentUsers(t *testing.T) {
	defer resetEaseeInstances()

	// Pre-populate two distinct entries.
	passwords := map[string]string{"user1@example.com": "pass1", "user2@example.com": "pass2"}
	for _, user := range []string{"user1@example.com", "user2@example.com"} {
		subject := easeeSubject(user, passwords[user])
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: user})
		easeeInstancesMu.Lock()
		easeeInstances[subject] = ts
		easeeInstancesMu.Unlock()
	}

	ts1, err := NewEaseeTokenSource("user1@example.com", "pass1")
	require.NoError(t, err)

	ts2, err := NewEaseeTokenSource("user2@example.com", "pass2")
	require.NoError(t, err)

	assert.NotEqual(t, ts1, ts2, "different users must have different token-sources")
}

// TestNewEaseeFromConfig_MissingCredentials verifies that omitting both
// user and password is rejected.
func TestNewEaseeFromConfig_MissingCredentials(t *testing.T) {
	_, err := newEaseeFromConfig(t.Context(), map[string]any{})
	require.ErrorIs(t, err, api.ErrMissingCredentials)
}

// TestNewEaseeFromConfig_MissingPassword verifies that a missing password is rejected.
func TestNewEaseeFromConfig_MissingPassword(t *testing.T) {
	_, err := newEaseeFromConfig(t.Context(), map[string]any{"user": "x@example.com"})
	require.ErrorIs(t, err, api.ErrMissingCredentials)
}

// TestNewEaseeFromConfig_MissingUser verifies that a missing user is rejected.
func TestNewEaseeFromConfig_MissingUser(t *testing.T) {
	_, err := newEaseeFromConfig(t.Context(), map[string]any{"password": "secret"})
	require.ErrorIs(t, err, api.ErrMissingCredentials)
}

// TestNewEaseeFromConfig_CacheHit verifies that a cached entry is returned
// when config is decoded successfully.
func TestNewEaseeFromConfig_CacheHit(t *testing.T) {
	defer resetEaseeInstances()

	user := "config@example.com"
	subject := easeeSubject(user, "pass")
	fakeTS := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "cached"})
	easeeInstancesMu.Lock()
	easeeInstances[subject] = fakeTS
	easeeInstancesMu.Unlock()

	ts, err := newEaseeFromConfig(t.Context(), map[string]any{
		"user":     user,
		"password": "pass",
	})
	require.NoError(t, err)
	assert.Equal(t, fakeTS, ts)
}

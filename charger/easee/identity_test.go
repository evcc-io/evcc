package easee

import (
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type fakeTokenSource struct {
	token *oauth2.Token
}

func (f *fakeTokenSource) Token() (*oauth2.Token, error) {
	return f.token, nil
}

func resetTokenSourceCache() {
	tokenSourceCache = cache.New[oauth2.TokenSource]()
}

// TestTokenSource_Deduplication verifies that the cache is keyed by account.
func TestTokenSource_Deduplication(t *testing.T) {
	t.Cleanup(resetTokenSourceCache)
	resetTokenSourceCache()

	account := "main-account"
	fakeTS := &fakeTokenSource{token: &oauth2.Token{AccessToken: "fake-at", RefreshToken: "fake-rt"}}

	_, err := tokenSourceCache.GetOrCreate(account, func() (oauth2.TokenSource, error) {
		return fakeTS, nil
	})
	require.NoError(t, err)

	log := util.NewLogger("easee-test")
	ts1, err := TokenSource(log, account, "user@example.com", "pass")
	require.NoError(t, err)
	require.NotNil(t, ts1)

	ts2, err := TokenSource(log, account, "user@example.com", "newpass")
	require.NoError(t, err)

	assert.Same(t, ts1, ts2, "same account must return the same cached token-source")
	assert.Same(t, fakeTS, ts1, "cached token-source must be reused")
}

// TestTokenSource_DifferentAccounts verifies that different accounts use distinct cache entries.
func TestTokenSource_DifferentAccounts(t *testing.T) {
	t.Cleanup(resetTokenSourceCache)
	resetTokenSourceCache()

	fakeTS1 := &fakeTokenSource{token: &oauth2.Token{AccessToken: "fake-at-1"}}
	fakeTS2 := &fakeTokenSource{token: &oauth2.Token{AccessToken: "fake-at-2"}}

	_, err := tokenSourceCache.GetOrCreate("account-1", func() (oauth2.TokenSource, error) {
		return fakeTS1, nil
	})
	require.NoError(t, err)

	_, err = tokenSourceCache.GetOrCreate("account-2", func() (oauth2.TokenSource, error) {
		return fakeTS2, nil
	})
	require.NoError(t, err)

	log := util.NewLogger("easee-test")
	ts1, err := TokenSource(log, "account-1", "user1@example.com", "pass1")
	require.NoError(t, err)

	ts2, err := TokenSource(log, "account-2", "user2@example.com", "pass2")
	require.NoError(t, err)

	assert.Same(t, fakeTS1, ts1)
	assert.Same(t, fakeTS2, ts2)
	assert.NotSame(t, ts1, ts2, "different accounts must have different token-sources")
}

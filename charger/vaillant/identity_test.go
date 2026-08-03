package vaillant

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/WulfgarW/sensonet"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// stubLogin replaces the real login and reports how often it was called
func stubLogin(t *testing.T) *int32 {
	t.Helper()

	tokens = make(map[string]oauth2.TokenSource)

	var calls int32
	loginFunc = func(_ context.Context, _ *util.Logger, _ *sensonet.Oauth2Config, user, _ string) (*oauth2.Token, error) {
		atomic.AddInt32(&calls, 1)
		return &oauth2.Token{AccessToken: user}, nil
	}

	t.Cleanup(func() {
		loginFunc = Login
		tokens = make(map[string]oauth2.TokenSource)
	})

	return &calls
}

func TestIdentityConcurrentSameAccountLogsInOnce(t *testing.T) {
	calls := stubLogin(t)

	oc := sensonet.Oauth2ConfigForRealm(sensonet.REALM_GERMANY)
	log := util.NewLogger("test")

	const n = 10
	var wg sync.WaitGroup
	ts := make([]oauth2.TokenSource, n)
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ts[i], errs[i] = Identity(log, oc, sensonet.REALM_GERMANY, "user@example.com", "secret")
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, *calls, "expected a single login for concurrent requests on the same account")

	for i := range n {
		require.True(t, ts[i] == ts[0], "expected all chargers to share the same token source")
	}
}

func TestIdentityDifferentAccountsLoginIndependently(t *testing.T) {
	calls := stubLogin(t)

	oc := sensonet.Oauth2ConfigForRealm(sensonet.REALM_GERMANY)
	log := util.NewLogger("test")

	_, err := Identity(log, oc, sensonet.REALM_GERMANY, "a@example.com", "secret")
	require.NoError(t, err)
	_, err = Identity(log, oc, sensonet.REALM_GERMANY, "b@example.com", "secret")
	require.NoError(t, err)

	require.EqualValues(t, 2, *calls, "expected independent logins for different accounts")
}

// the same user can exist in more than one realm, so the realm must be part of
// the cache key
func TestIdentitySameUserDifferentRealms(t *testing.T) {
	calls := stubLogin(t)

	oc := sensonet.Oauth2ConfigForRealm(sensonet.REALM_GERMANY)
	log := util.NewLogger("test")

	de, err := Identity(log, oc, sensonet.REALM_GERMANY, "user@example.com", "secret")
	require.NoError(t, err)
	at, err := Identity(log, oc, "vaillant-austria-b2c", "user@example.com", "secret")
	require.NoError(t, err)

	require.EqualValues(t, 2, *calls, "expected independent logins per realm")
	require.False(t, de == at, "expected separate token sources per realm")
}

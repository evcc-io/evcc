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

func TestIdentityConcurrentSameAccountLogsInOnce(t *testing.T) {
	tokens = make(map[string]oauth2.TokenSource)

	var calls int32
	loginFunc = func(_ context.Context, _ *util.Logger, _ *sensonet.Oauth2Config, user, _ string) (*oauth2.Token, error) {
		atomic.AddInt32(&calls, 1)
		return &oauth2.Token{AccessToken: user}, nil
	}
	t.Cleanup(func() { loginFunc = Login })

	oc := sensonet.Oauth2ConfigForRealm(sensonet.REALM_GERMANY)
	log := util.NewLogger("test")

	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = Identity(context.Background(), log, oc, "germany", "user@example.com", "secret")
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, calls, "expected a single login for concurrent requests on the same account")
}

func TestIdentityDifferentAccountsLoginIndependently(t *testing.T) {
	tokens = make(map[string]oauth2.TokenSource)

	var calls int32
	loginFunc = func(_ context.Context, _ *util.Logger, _ *sensonet.Oauth2Config, user, _ string) (*oauth2.Token, error) {
		atomic.AddInt32(&calls, 1)
		return &oauth2.Token{AccessToken: user}, nil
	}
	t.Cleanup(func() { loginFunc = Login })

	oc := sensonet.Oauth2ConfigForRealm(sensonet.REALM_GERMANY)
	log := util.NewLogger("test")

	_, err := Identity(context.Background(), log, oc, "germany", "a@example.com", "secret")
	require.NoError(t, err)
	_, err = Identity(context.Background(), log, oc, "germany", "b@example.com", "secret")
	require.NoError(t, err)

	require.EqualValues(t, 2, calls, "expected independent logins for different accounts")
}

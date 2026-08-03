package vaillant

import (
	"context"
	"sync"

	"github.com/WulfgarW/sensonet"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

var (
	mu     sync.Mutex
	tokens = make(map[string]oauth2.TokenSource)

	// loginFunc is a seam for testing; production code always uses Login
	loginFunc = Login
)

// Identity returns an oauth2 token source shared by all Vaillant chargers on the
// same myVaillant account, keyed by realm and user. The login is performed once
// while holding the lock, which serialises concurrent startups so the parallel
// login flows can no longer clobber each other's session; further chargers on the
// same account reuse the resulting refreshing token source (#30625).
func Identity(ctx context.Context, log *util.Logger, oc *sensonet.Oauth2Config, realm, user, password string) (oauth2.TokenSource, error) {
	mu.Lock()
	defer mu.Unlock()

	key := realm + "\x00" + user
	if ts, ok := tokens[key]; ok {
		return ts, nil
	}

	token, err := loginFunc(ctx, log, oc, user, password)
	if err != nil {
		return nil, err
	}

	ts := oc.TokenSource(ctx, token)
	tokens[key] = ts

	return ts, nil
}

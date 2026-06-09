package vaillant

import (
	"context"
	"sync"

	"github.com/WulfgarW/sensonet"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

var (
	mu         sync.Mutex
	identities = make(map[string]oauth2.TokenSource)
)

// Identity returns an oauth2 token source shared by all Vaillant chargers on the
// same myVaillant account, keyed by realm and user. The login is performed once
// while holding the lock, which serialises concurrent startups so the parallel
// Keycloak auth-code flows can no longer clobber each other; further chargers on
// the same account reuse the resulting refreshing token source (#30625).
func Identity(realm, user, password string) (oauth2.TokenSource, error) {
	mu.Lock()
	defer mu.Unlock()

	key := realm + "\x00" + user
	if ts, ok := identities[key]; ok {
		return ts, nil
	}

	log := util.NewLogger("vaillant").Redact(user, password)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(log))

	oc := sensonet.Oauth2ConfigForRealm(realm)
	token, err := oc.PasswordCredentialsToken(ctx, user, password)
	if err != nil {
		return nil, err
	}

	ts := oc.TokenSource(ctx, token)
	identities[key] = ts

	return ts, nil
}

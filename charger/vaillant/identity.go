package vaillant

import (
	"context"
	"strings"
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
// same myVaillant account. The login is performed once while holding the lock,
// which serialises concurrent startups so the parallel login flows can no longer
// clobber each other's session (#30625).
func Identity(log *util.Logger, oc *sensonet.Oauth2Config, realm, user, password string) (oauth2.TokenSource, error) {
	// serialise instance handling
	mu.Lock()
	defer mu.Unlock()

	// reuse identity instance
	subject := "vaillant." + strings.ToLower(realm) + "." + strings.ToLower(user)
	if ts, ok := identities[subject]; ok {
		return ts, nil
	}

	// decoupled from the calling charger's context, which is bounded by a timeout
	// during device creation and would otherwise cancel the shared token refresh
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(log))

	token, err := Login(ctx, log, oc, user, password)
	if err != nil {
		return nil, err
	}

	ts := oc.TokenSource(ctx, token)
	identities[subject] = ts

	return ts, nil
}

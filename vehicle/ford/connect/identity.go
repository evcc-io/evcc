package connect

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/plugin/auth"
	"golang.org/x/oauth2"
)

// https://api.vehicle.ford.com/fcon-public/v1/auth/init?client_id=799ef34f-99d3-45b2-939b-95f35abaa735&state=123456&redirect_uri=http://localhost:7070/providerauth/callback

func OAuth2Config(id, secret, redirectUri string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://api.vehicle.ford.com/fcon-public/v1/auth/init",
			TokenURL: "https://api.vehicle.ford.com/dah2vb2cprod.onmicrosoft.com/oauth2/v2.0/token",
		},
		RedirectURL: redirectUri,
		Scopes: []string{
			oidc.ScopeOpenID,
			oidc.ScopeOfflineAccess,
		},
	}
}

// NewIdentity creates FordConnect token source
func NewIdentity(id, secret, redirectUri string) (oauth2.TokenSource, error) {
	oc := OAuth2Config(id, secret, redirectUri)
	return auth.NewOauth(context.Background(), "Ford Connect", "", oc)
}

package connect

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
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
func NewIdentity(log *util.Logger, id, secret string, token *oauth2.Token) oauth2.TokenSource {
	oc := OAuth2Config(id, secret)
	client := request.NewClient(log)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
	return oc.TokenSource(ctx, token)
}

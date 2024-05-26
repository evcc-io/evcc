package connect

import (
	"context"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// ?p=B2C_1A_signup_signin_common

type Identity struct {
	*request.Helper
	oc *oauth2.Config
}

// NewIdentity creates autonomic token source
func NewIdentity(log *util.Logger, id, secret string, token *oauth2.Token) oauth2.TokenSource {
	oc := &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		Endpoint: oauth2.Endpoint{
			// AuthURL:  "https://accounts.autonomic.ai/v1/auth/oidc/authorize",
			TokenURL: "https://dah2vb2cprod.b2clogin.com/914d88b1-3523-4bf6-9be4-1b96b4f6f919/oauth2/v2.0/token",
		},
		RedirectURL: "http://localhost:3000",
		Scopes:      []string{"openid"},
	}

	return oc.TokenSource(context.Background(), token)
}

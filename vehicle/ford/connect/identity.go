package connect

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	ApplicationID = "AFDC085B-377A-4351-B23E-5E1D35FB3700"
	baseURL       = "https://dah2vb2cprod.b2clogin.com/914d88b1-3523-4bf6-9be4-1b96b4f6f919/oauth2/v2.0/token?p=B2C_1A_signup_signin_common"
)

func Oauth2Config(id, secret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  baseURL,
			TokenURL: baseURL,
		},
		RedirectURL: "https://localhost:3000",
		Scopes: []string{
			oidc.ScopeOpenID,
			oidc.ScopeOfflineAccess,
		},
	}
}

// NewIdentity creates FordConnect token source
func NewIdentity(log *util.Logger, id, secret string, token *oauth2.Token) oauth2.TokenSource {
	oc := Oauth2Config(id, secret)
	client := request.NewClient(log)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
	return oc.TokenSource(ctx, token)
}

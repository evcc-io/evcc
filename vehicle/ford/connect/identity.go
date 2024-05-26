package connect

import (
	"context"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type Identity struct {
	*request.Helper
	oc *oauth2.Config
}

const baseURL = "https://dah2vb2cprod.b2clogin.com/914d88b1-3523-4bf6-9be4-1b96b4f6f919/oauth2/v2.0/token?p=B2C_1A_signup_signin_common"

func Oauth2Config(id, secret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  baseURL,
			TokenURL: baseURL,
		},
		RedirectURL: "https://localhost:3000",
		Scopes:      []string{"openid"},
	}
}

// NewIdentity creates autonomic token source
func NewIdentity(log *util.Logger, id, secret string, token *oauth2.Token) oauth2.TokenSource {
	oc := Oauth2Config(id, secret)
	return oc.TokenSource(context.Background(), token)
}

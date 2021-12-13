package psa

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/util/logx"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type Identity struct {
	*request.Helper
	oc *oauth2.Config
	oauth2.TokenSource
}

// NewIdentity creates PSA identity
func NewIdentity(log logx.Logger, brand, id, secret string) *Identity {
	return &Identity{
		Helper: request.NewHelper(log),
		oc: &oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://api.mpsa.com/api/connectedcar/v2/oauth/authorize",
				TokenURL:  fmt.Sprintf("https://idpcvs.%s/am/oauth2/access_token", brand),
				AuthStyle: oauth2.AuthStyleInHeader,
			},
			Scopes: []string{"openid profile"},
		},
	}
}

func (v *Identity) Login(user, password string) error {
	ctx := context.WithValue(
		context.Background(),
		oauth2.HTTPClient,
		v.Client,
	)

	// replace client with authenticated oauth client
	token, err := v.oc.PasswordCredentialsToken(ctx, user, password)
	if err == nil {
		v.TokenSource = v.oc.TokenSource(ctx, token)
	}

	return err
}

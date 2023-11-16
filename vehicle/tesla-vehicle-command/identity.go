package vc

import (
	"os"

	"github.com/evcc-io/evcc/util"
	"github.com/teslamotors/vehicle-command/pkg/account"
	"golang.org/x/oauth2"
)

// https://auth.tesla.com/oauth2/v3/.well-known/openid-configuration

// OAuth2Config is the OAuth2 configuration for authenticating with the Tesla API.
var OAuth2Config = &oauth2.Config{
	ClientID:    os.Getenv("TESLA_CLIENT_ID"),
	RedirectURL: "https://auth.tesla.com/void/callback",
	Endpoint: oauth2.Endpoint{
		AuthURL:   "https://auth.tesla.com/en_us/oauth2/v3/authorize",
		TokenURL:  "https://auth.tesla.com/oauth2/v3/token",
		AuthStyle: oauth2.AuthStyleInParams,
	},
	Scopes: []string{"openid", "email", "offline_access"},
}

type Identity struct {
	log   *util.Logger
	ts    oauth2.TokenSource
	token *oauth2.Token
	acct  *account.Account
}

func NewIdentity(log *util.Logger, ts oauth2.TokenSource) (*Identity, error) {
	token, err := ts.Token()
	if err != nil {
		return nil, err
	}

	acct, err := account.New(token.AccessToken)
	if err != nil {
		return nil, err
	}

	return &Identity{
		ts:    ts,
		token: token,
		acct:  acct,
	}, nil
}

func (v *Identity) Account() *account.Account {
	token, err := v.ts.Token()
	if err != nil {
		v.log.ERROR.Println(err)
		return v.acct
	}

	if token.AccessToken != v.token.AccessToken {
		acct, err := account.New(token.AccessToken)
		if err != nil {
			v.log.ERROR.Println(err)
			return v.acct
		}

		v.acct = acct
	}

	return v.acct
}

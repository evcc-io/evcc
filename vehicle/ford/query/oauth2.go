package query

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

func init() {
	auth.Register("ford-connect", func(other map[string]any) (oauth2.TokenSource, error) {
		var cc struct {
			ClientID     string
			ClientSecret string
			RedirectURI  string
		}

		if err := util.DecodeOther(other, &cc); err != nil {
			return nil, err
		}

		oc := OAuth2Config(cc.ClientID, cc.ClientSecret, cc.RedirectURI)

		return NewOAuth(oc, "")
	})
}

func OAuth2Config(id, secret, redirectUri string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://api.vehicle.ford.com/fcon-public/v1/auth/init",
			TokenURL:  "https://api.vehicle.ford.com/dah2vb2cprod.onmicrosoft.com/oauth2/v2.0/token?p=B2C_1A_FCON_AUTHORIZE",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: redirectUri,
		Scopes: []string{
			oidc.ScopeOpenID,
			oidc.ScopeOfflineAccess,
		},
	}
}

// NewOAuth creates FordConnect token source
func NewOAuth(oc *oauth2.Config, title string) (oauth2.TokenSource, error) {
	return auth.NewOAuth(context.Background(), "Ford Connect", title, oc)
}

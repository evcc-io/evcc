package connected

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

func init() {
	auth.Register("volvo-connected", func(other map[string]any) (oauth2.TokenSource, error) {
		var cc struct {
			ClientID     string
			ClientSecret string
			RedirectUri  string
		}

		if err := util.DecodeOther(other, &cc); err != nil {
			return nil, err
		}

		oc := OAuthConfig(cc.ClientID, cc.ClientSecret, cc.RedirectUri)

		return NewOAuth(oc, "")
	})
}

func OAuthConfig(id, secret, redirectUri string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		RedirectURL:  redirectUri,
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://volvoid.eu.volvocars.com/as/authorization.oauth2",
			TokenURL:  "https://volvoid.eu.volvocars.com/as/token.oauth2",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		Scopes: []string{
			oidc.ScopeOpenID,
			"conve:vehicle_relation",
			"energy:state:read",
			"conve:odometer_status",
		},
	}
}

func NewOAuth(oc *oauth2.Config, title string) (oauth2.TokenSource, error) {
	return auth.NewOAuth(context.Background(), "Volvo", title, oc)
}

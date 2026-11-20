package connected

import (
	"context"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

var (
	endpoint = oauth2.Endpoint{
		AuthURL:   "https://auth.volvocars.com/realms/volvocars/protocol/openid-connect/auth",
		TokenURL:  "https://auth.volvocars.com/realms/volvocars/protocol/openid-connect/token",
		AuthStyle: oauth2.AuthStyleInHeader,
	}

	// TODO remove once Volvo shuts down the legacy endpoint on 2027-01-31
	legacyEndpoint = oauth2.Endpoint{
		AuthURL:   "https://volvoid.eu.volvocars.com/as/authorization.oauth2",
		TokenURL:  "https://volvoid.eu.volvocars.com/as/token.oauth2",
		AuthStyle: oauth2.AuthStyleInHeader,
	}
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

		log := util.NewLogger("volvo").Redact(cc.ClientID)
		oc := OAuthConfig(cc.ClientID, cc.ClientSecret, cc.RedirectUri)

		return NewOAuth(util.WithLogger(context.Background(), log), oc, "")
	})
}

// isLegacyClient reports whether the client id belongs to an application created before Volvo's OAuth2 migration
func isLegacyClient(id string) bool {
	return strings.HasPrefix(id, "dc-")
}

// OAuthConfig returns the OAuth2 config, selecting the legacy or current endpoint by client id
func OAuthConfig(id, secret, redirectUri string) *oauth2.Config {
	ep := endpoint
	if isLegacyClient(id) {
		ep = legacyEndpoint
		util.NewLogger("volvo").WARN.Println("legacy Volvo developer application detected, it will stop working on 31 January 2027. Create and publish a new application at https://developer.volvocars.com/account/ and update client id, client secret and VCC API key")
	}

	return &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		RedirectURL:  redirectUri,
		Endpoint:     ep,
		Scopes: []string{
			oidc.ScopeOpenID,
			"conve:vehicle_relation",
			"energy:state:read",
			"conve:odometer_status",
		},
	}
}

func NewOAuth(ctx context.Context, oc *oauth2.Config, title string) (oauth2.TokenSource, error) {
	return auth.NewOAuth(ctx, "Volvo", title, oc)
}

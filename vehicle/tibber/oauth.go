package tibber

import (
	"context"

	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

const (
	// AuthURI is the Tibber Data API authorization endpoint.
	AuthURI = "https://thewall.tibber.com/connect/authorize"
	// TokenURI is the Tibber Data API token endpoint.
	TokenURI = "https://thewall.tibber.com/connect/token"
	// ApiURI is the Tibber Data API base URL.
	ApiURI = "https://data-api.tibber.com/v1"
)

func init() {
	auth.Register("tibber", func(other map[string]any) (oauth2.TokenSource, error) {
		var cc struct {
			ClientID, ClientSecret, RedirectURI string
		}

		if err := util.DecodeOther(other, &cc); err != nil {
			return nil, err
		}

		log := util.NewLogger("tibber").Redact(cc.ClientID, cc.ClientSecret)
		ctx := util.WithLogger(context.Background(), log)

		return NewOAuth(ctx, cc.ClientID, cc.ClientSecret, cc.RedirectURI, "")
	})
}

// OAuthConfig returns the Tibber Data API OAuth2 config.
func OAuthConfig(clientID, clientSecret, redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Endpoint: oauth2.Endpoint{
			AuthURL:  AuthURI,
			TokenURL: TokenURI,
		},
		Scopes: []string{
			"openid", "profile", "email", "offline_access",
			"data-api-user-read", "data-api-vehicles-read",
		},
	}
}

// NewOAuth creates the Tibber Data API token source using the authorization
// code flow with PKCE. The user authorizes interactively via the evcc UI.
func NewOAuth(ctx context.Context, clientID, clientSecret, redirectURI, title string) (oauth2.TokenSource, error) {
	return auth.NewOAuth(ctx, "Tibber", title, OAuthConfig(clientID, clientSecret, redirectURI))
}

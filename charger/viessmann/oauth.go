package viessmann

import (
	"context"

	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

const (
	// OAuthURI is the Viessmann IAM base URL.
	OAuthURI = "https://iam.viessmann-climatesolutions.com/idp/v3"
	// ApiURI is the Viessmann IoT API base URL.
	ApiURI = "https://api.viessmann-climatesolutions.com/iot/v2"
)

func init() {
	auth.Register("viessmann", func(other map[string]any) (oauth2.TokenSource, error) {
		var cc struct {
			ClientID    string
			RedirectURI string
			Gateway     string `mapstructure:"gateway_serial"`
		}

		if err := util.DecodeOther(other, &cc); err != nil {
			return nil, err
		}

		log := util.NewLogger("viessmann").Redact(cc.ClientID)
		ctx := util.WithLogger(context.Background(), log)

		return NewOAuth(ctx, cc.ClientID, cc.RedirectURI, cc.Gateway)
	})
}

// OAuthConfig returns the Viessmann IoT API OAuth2 config.
func OAuthConfig(clientID, redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:    clientID,
		RedirectURL: redirectURI,
		Endpoint: oauth2.Endpoint{
			AuthURL:   OAuthURI + "/authorize",
			TokenURL:  OAuthURI + "/token",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		Scopes: []string{"IoT User", "offline_access"},
	}
}

// NewOAuth creates the Viessmann IoT API token source using the authorization
// code flow. The user authorizes interactively via the evcc UI.
func NewOAuth(ctx context.Context, clientID, redirectURI, device string) (oauth2.TokenSource, error) {
	return auth.NewOAuth(ctx, "Viessmann", device, OAuthConfig(clientID, redirectURI))
}

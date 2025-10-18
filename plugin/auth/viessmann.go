package auth

import (
	"context"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const (
	OAuthURI    = "https://iam.viessmann-climatesolutions.com/idp/v3"
	RedirectURI = "http://localhost:4200/"
	// ^ the value of RedirectURI doesn't matter, but it must be the same between requests
)

func oauth2Config(clientID string) *oauth2.Config {
	return &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:   OAuthURI + "/authorize",
			TokenURL:  OAuthURI + "/token",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		RedirectURL: RedirectURI,
		Scopes:      []string{"IoT User", "offline_access"},
	}
}

func init() {
	registry.AddCtx("viessmann", NewViessmannFromConfig)
}

func NewViessmannFromConfig(ctx context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		ClientID string
		Gateway  string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("viessmann").Redact(cc.ClientID)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, request.NewClient(log))

	return NewOauth(ctx, "Viessmann", cc.Gateway, oauth2Config(cc.ClientID))
}

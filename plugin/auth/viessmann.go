package auth

import (
	"context"

	"dario.cat/mergo"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

const OAuthURI = "https://iam.viessmann-climatesolutions.com/idp/v3"

var oauthConfig = oauth2.Config{
	Endpoint: oauth2.Endpoint{
		AuthURL:   OAuthURI + "/authorize",
		TokenURL:  OAuthURI + "/token",
		AuthStyle: oauth2.AuthStyleInHeader,
	},
	Scopes: []string{"IoT User", "offline_access"},
}

func init() {
	registry.AddCtx("viessmann", NewViessmannFromConfig)
}

func NewViessmannFromConfig(ctx context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		ClientID    string
		RedirectURI string
		Gateway     string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("viessmann").Redact(cc.ClientID)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, request.NewClient(log))

	oc := oauth2.Config{
		ClientID:    cc.ClientID,
		RedirectURL: cc.RedirectURI,
	}
	if err := mergo.Merge(&oc, oauthConfig); err != nil {
		return nil, err
	}

	return NewOauth(ctx, "Viessmann", cc.Gateway, &oc)
}

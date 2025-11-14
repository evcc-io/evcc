package auth

import (
	"context"

	"github.com/evcc-io/evcc/server/network"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// https://developers.home-assistant.io/docs/auth_api

func init() {
	registry.AddCtx("homeassistant", NewHomeAssistantFromConfig)
}

func NewHomeAssistantFromConfig(ctx context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		URI  string
		Name string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHomeAssistant(ctx, cc.Name, cc.URI)
}

func NewHomeAssistant(ctx context.Context, name, uri string) (*OAuth, error) {
	extUrl := network.Config().ExternalURL()
	redirectUri := extUrl + "/providerauth/callback"

	log := util.NewLogger("homeassistant")
	ctx = context.WithValue(ctx, oauth2.HTTPClient, request.NewClient(log))

	oc := oauth2.Config{
		ClientID:    extUrl,
		RedirectURL: redirectUri,
		Endpoint: oauth2.Endpoint{
			AuthURL:  uri + "/auth/authorize",
			TokenURL: uri + "/auth/token",
		},
	}

	return NewOauth(ctx, "HomeAssistant", name, &oc)
}

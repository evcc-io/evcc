package homeassistant

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/server/network"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// https://developers.home-assistant.io/docs/auth_api

func init() {
	auth.Register("homeassistant", NewHomeAssistantFromConfig)
}

func NewHomeAssistantFromConfig(other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		Home string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	inst := instanceByName(cc.Home)
	if inst == nil {
		return nil, fmt.Errorf("unknown instance: %s", cc.Home)
	}

	return NewHomeAssistant(cc.Home, inst.URI)
}

func NewHomeAssistant(name, uri string) (oauth2.TokenSource, error) {
	extUrl := network.Config().ExternalURL()
	redirectUri := extUrl + "/providerauth/callback"

	log := util.NewLogger("homeassistant")
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(log))

	oc := oauth2.Config{
		ClientID:    extUrl,
		RedirectURL: redirectUri,
		Endpoint: oauth2.Endpoint{
			AuthURL:  uri + "/auth/authorize",
			TokenURL: uri + "/auth/token",
		},
	}

	return auth.NewOauth(ctx, "HomeAssistant", name, &oc)
}

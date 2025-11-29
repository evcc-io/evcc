package homeassistant

import (
	"context"
	"fmt"
	"net/url"

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
		URI  string
		Home string // TODO deprecate, backward compatibility (v0.210.x)
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	uri := cc.URI

	if uri == "" && cc.Home != "" {
		uri = instanceUriByName(cc.Home)
		if uri == "" {
			return nil, fmt.Errorf("unknown instance: %s", cc.Home)
		}
	}

	if uri == "" {
		return nil, fmt.Errorf("missing uri parameter")
	}

	return NewHomeAssistant(uri)
}

func NewHomeAssistant(uri string) (oauth2.TokenSource, error) {
	extUrl := network.Config().ExternalURL()
	redirectUri := extUrl + network.CallbackPath

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

	// use host as device name
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	return auth.NewOAuth(ctx, "HomeAssistant", u.Host, &oc)
}

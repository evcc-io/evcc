package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

func init() {
	registry.AddCtx("homeassistant", NewHomeAssistantFromConfig)

	if _, err := NewHomeAssistantFromConfig(context.Background(), map[string]any{
		"uri": "http://localhost:8123",
	}); err == nil {
		fmt.Println("HomeAssistant configured")
	} else {
		fmt.Println(err)
	}
}

func NewHomeAssistantFromConfig(ctx context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		URI string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	instance := util.DefaultScheme(strings.TrimSuffix(cc.URI, "/"), "http")

	uri := "http://localhost:7070"
	redirectUri := uri + "/providerauth/callback"

	log := util.NewLogger("homeassistant")
	ctx = context.WithValue(ctx, oauth2.HTTPClient, request.NewClient(log))

	oc := oauth2.Config{
		ClientID:    uri,
		RedirectURL: redirectUri,
		Endpoint: oauth2.Endpoint{
			AuthURL:  instance + "/auth/authorize",
			TokenURL: instance + "/auth/token",
		},
	}

	return NewOauth(ctx, "HomeAssistant", instance, &oc)
}

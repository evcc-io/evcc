package auth

import (
	"context"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

func init() {
	registry.AddCtx("homeassistant", NewHomeAssistantFromConfig)

	// if _, err := NewHomeAssistantFromConfig(context.Background(), map[string]any{
	// 	"uri": "http://localhost:8123",
	// }); err == nil {
	// 	fmt.Println("HomeAssistant configured")
	// } else {
	// 	fmt.Println(err)
	// }
}

func NewHomeAssistantFromConfig(ctx context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		URI  string
		Name string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	instanceUri := util.DefaultScheme(strings.TrimSuffix(cc.URI, "/"), "http")
	instanceName := cc.Name
	if instanceName == "" {
		instanceName = instanceUri
		if uri, err := url.Parse(instanceUri); err == nil {
			instanceName = uri.Host
		}
	}

	uri := "http://localhost:7070"
	redirectUri := uri + "/providerauth/callback"

	log := util.NewLogger("homeassistant")
	ctx = context.WithValue(ctx, oauth2.HTTPClient, request.NewClient(log))

	oc := oauth2.Config{
		ClientID:    uri,
		RedirectURL: redirectUri,
		Endpoint: oauth2.Endpoint{
			AuthURL:  instanceUri + "/auth/authorize",
			TokenURL: instanceUri + "/auth/token",
		},
	}

	return NewOauth(ctx, "HomeAssistant", instanceName, &oc)
}

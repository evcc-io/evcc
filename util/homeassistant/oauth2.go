package homeassistant

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/server/network"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

// https://developers.home-assistant.io/docs/auth_api

// NewOAuth creates a Home Assistant OAuth token source
func NewOAuth(uri string, insecure bool) (oauth2.TokenSource, error) {
	uri = strings.TrimRight(uri, "/") // normalize

	extUrl := network.Config().ExternalURL()
	redirectUri := extUrl + network.CallbackPath

	oc := oauth2.Config{
		ClientID:    extUrl,
		RedirectURL: redirectUri,
		Endpoint: oauth2.Endpoint{
			AuthURL:   uri + "/auth/authorize",
			TokenURL:  uri + "/auth/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}

	// validate url
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	host := u.Host
	if h, _, err := net.SplitHostPort(u.Host); err == nil {
		host = h
	}

	// use instance name instead of host if discovered on mDNS
	if name := instanceNameByUri(uri); name != "" {
		host = name
	}

	log := util.NewLogger("homeassistant")
	ctx := util.WithLogger(context.Background(), log)

	if insecure {
		log.WARN.Println("insecure mode enabled - TLS certificate verification is disabled, use only for trusted local/self-signed instances")
		httpClient := &http.Client{Transport: transport.Insecure()}
		ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	}

	return auth.NewOAuth(ctx, "HomeAssistant", host, &oc)
}

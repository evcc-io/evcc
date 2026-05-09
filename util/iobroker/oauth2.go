package iobroker

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/server/network"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

func init() {
	auth.Register("iobroker", NewIobrokerFromConfig)
}

func NewIobrokerFromConfig(other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		URI string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	uri := cc.URI

	if uri == "" {
		return nil, fmt.Errorf("URI missing")
	}

	return NewIobroker(uri)
}

func NewIobroker(uri string) (oauth2.TokenSource, error) {
	uri = strings.TrimRight(uri, "/") // normalize

	extUrl := network.Config().ExternalURL()
	redirectUri := extUrl + network.CallbackPath

	oc := oauth2.Config{
		ClientID:    "iobroker",
		RedirectURL: redirectUri,
		Endpoint: oauth2.Endpoint{
			AuthURL:   uri + "/oauth/token",
			TokenURL:  uri + "/oauth/token",
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

	log := util.NewLogger("iobroker")
	ctx := util.WithLogger(context.Background(), log)

	return auth.NewOAuth(ctx, "Iobroker", host, &oc)
}

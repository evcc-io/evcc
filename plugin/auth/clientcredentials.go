package auth

import (
	"context"

	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

func init() {
	registry.AddCtx("clientcredentials", NewClientcredentialsFromConfig)
}

func NewClientcredentialsFromConfig(ctx context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		ClientID     string
		ClientSecret string
		TokenURL     string
		Scopes       []string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	conf := &clientcredentials.Config{
		ClientID:     cc.ClientID,
		ClientSecret: cc.ClientSecret,
		TokenURL:     cc.TokenURL,
		Scopes:       cc.Scopes,
	}

	ts := conf.TokenSource(ctx)

	return ts, nil
}

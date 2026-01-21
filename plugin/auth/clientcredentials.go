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
	var conf clientcredentials.Config

	if err := util.DecodeOther(other, &conf); err != nil {
		return nil, err
	}
	ts := conf.TokenSource(ctx)

	return ts, nil
}

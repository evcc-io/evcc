package auth

import (
	"context"
	"errors"

	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

func init() {
	registry.AddCtx("clientcredentials", NewClientcredentialsFromConfig)
}

func NewClientcredentialsFromConfig(ctx context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc clientcredentials.Config

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	switch {
	case cc.ClientID == "":
		return nil, errors.New("clientcredentials: missing required clientid")
	case cc.ClientSecret == "":
		return nil, errors.New("clientcredentials: missing required clientsecret")
	case cc.TokenURL == "":
		return nil, errors.New("clientcredentials: missing required tokenurl")
	}

	return cc.TokenSource(ctx), nil
}

package auth

import (
	"context"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/providerauth"
	"golang.org/x/oauth2"
)

func init() {
	registry.AddCtx("demo", NewDemoFromConfig)
}

type demo struct {
	token *oauth2.Token
}

func NewDemoFromConfig(_ context.Context, _ map[string]any) (oauth2.TokenSource, error) {
	return NewDemo()
}

func NewDemo() (oauth2.TokenSource, error) {
	o := new(demo)

	// register auth redirect
	if _, err := providerauth.Register("demo", o); err != nil {
		return nil, err
	}

	return o, nil
}

func (o *demo) Token() (*oauth2.Token, error) {
	var err error
	if o.token == nil {
		err = api.LoginRequiredError("demo")
	}
	return o.token, err
}

func (o *demo) Login(_ string) (string, error) {
	// TODO use network settings after https://github.com/evcc-io/evcc/pull/25141
	return "http://localhost:7070/providerauth/callback", nil
}

func (o *demo) Logout() error {
	o.token = nil
	return nil
}

func (o *demo) HandleCallback(params url.Values) error {
	o.token = &oauth2.Token{
		AccessToken: "foo",
		Expiry:      time.Now().Add(24 * time.Hour),
	}
	return nil
}

func (o *demo) Authenticated() bool {
	return o.token != nil
}

func (o *demo) DisplayName() string {
	return "demo"
}

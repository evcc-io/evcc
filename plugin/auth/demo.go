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

var demoInstance *demo

func NewDemoFromConfig(_ context.Context, _ map[string]any) (oauth2.TokenSource, error) {
	return NewDemo()
}

func NewDemo() (oauth2.TokenSource, error) {
	// reuse instance (similar to oauth.go getInstance pattern)
	if demoInstance != nil {
		return demoInstance, nil
	}

	demoInstance = new(demo)

	if _, err := providerauth.Register("demo", demoInstance); err != nil {
		return nil, err
	}

	return demoInstance, nil
}

func (o *demo) Token() (*oauth2.Token, error) {
	if o.token == nil {
		return nil, api.LoginRequiredError("demo")
	}
	return o.token, nil
}

func (o *demo) Login(_ string) (string, error) {
	// for demo, immediately authenticate without requiring external flow
	o.token = &oauth2.Token{
		AccessToken: "demo-token",
		Expiry:      time.Now().Add(24 * time.Hour),
	}
	// TODO use network settings after https://github.com/evcc-io/evcc/pull/25141
	return "http://localhost:7070/providerauth/callback", nil
}

func (o *demo) Logout() error {
	o.token = nil
	return nil
}

func (o *demo) HandleCallback(params url.Values) error {
	// no-op: token already set in Login()
	return nil
}

func (o *demo) Authenticated() bool {
	return o.token != nil
}

func (o *demo) DisplayName() string {
	return "demo"
}

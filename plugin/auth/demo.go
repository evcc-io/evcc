package auth

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/providerauth"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

func init() {
	registry.AddCtx("demo", NewDemoFromConfig)
}

type demo struct {
	token       *oauth2.Token
	server      string
	method      string
	redirectUri string
}

var demoInstance *demo

func NewDemoFromConfig(_ context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		Server      string
		Method      string
		RedirectUri string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDemo(cc.Server, cc.Method, cc.RedirectUri)
}

func NewDemo(server string, method string, redirectUri string) (oauth2.TokenSource, error) {
	// reuse instance (similar to oauth.go getInstance pattern)
	if demoInstance != nil {
		// update existing instance with new values
		demoInstance.server = server
		demoInstance.method = method
		demoInstance.redirectUri = redirectUri
		return demoInstance, nil
	}

	demoInstance = &demo{
		server:      server,
		method:      method,
		redirectUri: redirectUri,
	}

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

func (o *demo) Login(state string) (string, *oauth2.DeviceAuthResponse, error) {
	// Simulate error for ERROR server
	if o.server == "ERROR" {
		return "", nil, fmt.Errorf("server not supported")
	}

	// Build mock login URL with state and redirectUri parameters
	mockLoginURL := fmt.Sprintf("%s/mock-login?state=%s&redirectUri=%s", o.server, state, o.redirectUri)

	if o.method == "device-code" {
		// Device code flow: URI comes from DeviceAuthResponse
		return "", &oauth2.DeviceAuthResponse{
			UserCode:        "12AB345",
			VerificationURI: mockLoginURL,
			Expiry:          time.Now().Add(10 * time.Minute),
		}, nil
	}

	// Redirect flow: URI in first return value
	return mockLoginURL, nil, nil
}

func (o *demo) Logout() error {
	o.token = nil
	return nil
}

func (o *demo) HandleCallback(params url.Values) error {
	// Extract code from callback parameters
	code := params.Get("code")
	if code == "" {
		return fmt.Errorf("missing code parameter")
	}

	// Create token based on code (for demo, we use a fixed token)
	o.token = &oauth2.Token{
		AccessToken: code, // Use the code as the access token
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

package auth

import (
	"context"
	"fmt"
	"net/url"
	"strings"
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
	onlineC     chan<- bool
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

	onlineC, err := providerauth.Register("demo", demoInstance)
	if err != nil {
		return nil, err
	}
	demoInstance.onlineC = onlineC

	// Send initial auth status
	demoInstance.onlineC <- false

	return demoInstance, nil
}

func (o *demo) Token() (*oauth2.Token, error) {
	if o.token == nil {
		return nil, api.LoginRequiredError("demo")
	}
	return o.token, nil
}

func (o *demo) Login(state string) (string, *oauth2.DeviceAuthResponse, error) {
	// Validate server URL has proper scheme
	if !strings.HasPrefix(o.server, "http://") && !strings.HasPrefix(o.server, "https://") {
		return "", nil, fmt.Errorf("server must start with http:// or https://")
	}

	// Validate redirect URI has proper scheme
	if !strings.HasPrefix(o.redirectUri, "http://") && !strings.HasPrefix(o.redirectUri, "https://") {
		return "", nil, fmt.Errorf("redirectUri must start with http:// or https://")
	}

	// Build mock login URL with state and redirectUri (complete callback URL)
	values := url.Values{}
	values.Set("state", state)
	values.Set("redirectUri", o.redirectUri)

	mockLoginURL := fmt.Sprintf("%s/mock-login?%s", o.server, values.Encode())

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
	if o.onlineC != nil {
		o.onlineC <- false
	}
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

	// Notify that authentication succeeded
	if o.onlineC != nil {
		o.onlineC <- true
	}

	return nil
}

func (o *demo) Authenticated() bool {
	return o.token != nil
}

func (o *demo) DisplayName() string {
	return "Demo Auth"
}

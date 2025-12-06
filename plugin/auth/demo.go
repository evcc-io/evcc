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
	token  *oauth2.Token
	region string
	method string
}

var demoInstance *demo

func NewDemoFromConfig(_ context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		Region string
		Method string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDemo(cc.Region, cc.Method)
}

func NewDemo(region string, method string) (oauth2.TokenSource, error) {
	// reuse instance (similar to oauth.go getInstance pattern)
	if demoInstance != nil {
		// update existing instance with new values
		demoInstance.region = region
		demoInstance.method = method
		return demoInstance, nil
	}

	demoInstance = &demo{
		region: region,
		method: method,
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

func (o *demo) Login(_ string) (string, *oauth2.DeviceAuthResponse, error) {
	// Simulate error for ERROR region
	if o.region == "ERROR" {
		return "", nil, fmt.Errorf("region not supported")
	}

	// For demo, immediately authenticate without requiring external flow
	o.token = &oauth2.Token{
		AccessToken: "demo-token",
		Expiry:      time.Now().Add(24 * time.Hour),
	}

	// TODO use network settings after https://github.com/evcc-io/evcc/pull/25141
	uri := "http://localhost:7070/providerauth/callback"

	if o.method == "device-code" {
		// Device code flow: URI comes from DeviceAuthResponse
		return "", &oauth2.DeviceAuthResponse{
			UserCode:        "12AB345",
			VerificationURI: uri,
			Expiry:          time.Now().Add(10 * time.Minute),
		}, nil
	}

	// Redirect flow: URI in first return value
	return uri, nil, nil
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

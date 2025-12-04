package kamereon

import (
	"net/http"

	"github.com/evcc-io/evcc/vehicle/renault/gigya"
	"github.com/evcc-io/evcc/vehicle/renault/keys"
)

type AuthDecorator struct {
	Base     http.RoundTripper
	Login    func() error
	Keys     keys.ConfigServer
	Identity *gigya.Identity
}

func (rt *AuthDecorator) RoundTrip(req *http.Request) (*http.Response, error) {
	// Set required headers
	req.Header.Set("content-type", "application/vnd.api+json")
	req.Header.Set("x-gigya-id_token", rt.Identity.Token)
	req.Header.Set("apikey", rt.Keys.APIKey)

	// Set country query parameter
	q := req.URL.Query()
	q.Set("country", "DE")
	req.URL.RawQuery = q.Encode()

	resp, err := rt.Base.RoundTrip(req)
	if err == nil {
		return resp, nil
	}

	// Try reauthenticating
	if err := rt.Login(); err != nil {
		return resp, err
	}

	// Retry the request
	return rt.Base.RoundTrip(req)
}

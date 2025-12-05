package kamereon

import (
	"bytes"
	"io"
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
	// Buffer request body for potential retries
	var (
		bodyBuffer []byte
		err        error
	)
	if req.Body != nil {
		bodyBuffer, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		_ = req.Body.Close()

		req.Body = io.NopCloser(bytes.NewReader(bodyBuffer))
	}

	resp, err := rt.executeRequest(req)

	if err == nil && resp != nil && resp.StatusCode == http.StatusUnauthorized {
		// Drain and close response body
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// Try reauthenticating
		if err := rt.Login(); err != nil {
			return nil, err
		}

		// Reset request body
		if bodyBuffer != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBuffer))
		}

		// Retry the request
		resp, err = rt.executeRequest(req)
	}

	return resp, err
}

func (rt *AuthDecorator) executeRequest(req *http.Request) (*http.Response, error) {
	// Set required headers
	req.Header.Set("content-type", "application/vnd.api+json")
	req.Header.Set("x-gigya-id_token", rt.Identity.Token)
	req.Header.Set("apikey", rt.Keys.APIKey)

	// Set country query parameter
	q := req.URL.Query()
	q.Set("country", "DE")
	req.URL.RawQuery = q.Encode()

	base := rt.Base
	if base == nil {
		base = http.DefaultTransport
	}

	return base.RoundTrip(req)
}

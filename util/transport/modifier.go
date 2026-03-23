package transport

import (
	"net/http"
)

// Modifier is an http.RoundTripper that makes HTTP responses,
// wrapping a base RoundTripper and modifying given responses.
type Modifier struct {
	// Modifier modifies the incoming response
	Modifier func(*http.Response) error

	// Base is the base RoundTripper used to make HTTP responses.
	Base http.RoundTripper
}

// RoundTrip modifies the response using the Modifier.
func (t *Modifier) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.Base.RoundTrip(req)
	if err != nil || t.Modifier == nil {
		return resp, err
	}

	err = t.Modifier(resp)
	return resp, err
}

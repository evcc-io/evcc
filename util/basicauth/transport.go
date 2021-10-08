package basicauth

import (
	"encoding/base64"
	"net/http"
)

type transport struct {
	header string
	base   http.RoundTripper
}

// Header returns the basic auth header
func Header(user, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
}

// NewTransport creates an http transport performing basic auth
func NewTransport(user, password string, base http.RoundTripper) http.RoundTripper {
	return &transport{
		header: Header(user, password),
		base:   base,
	}
}

// RoundTrip implements the http.RoundTripper interface
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.header)
	return t.base.RoundTrip(req)
}

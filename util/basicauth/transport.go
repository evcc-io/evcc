package basicauth

import (
	"encoding/base64"
	"net/http"
)

type transport struct {
	header string
	base   http.RoundTripper
}

// NewTransport creates an http transport performing basic auth
func NewTransport(user, password string, base http.RoundTripper) http.RoundTripper {
	return &transport{
		header: "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password)),
		base:   base,
	}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.header)
	return t.base.RoundTrip(req)
}

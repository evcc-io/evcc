package transport

import (
	"encoding/base64"
	"net/http"
)

// BasicAuthHeader returns the basic auth header
func BasicAuthHeader(user, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
}

// BasicAuth creates an http transport performing basic auth
func BasicAuth(user, password string, base http.RoundTripper) http.RoundTripper {
	return &Decorator{
		Decorator: DecorateHeaders(map[string]string{
			"Authorization": BasicAuthHeader(user, password),
		}),
		Base: base,
	}
}

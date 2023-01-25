package transport

import (
	"net/http"
)

// BearerAuth creates an HTTP transport performing HTTP authorization using an OAuth 2.0 Bearer Token
func BearerAuth(token string, base http.RoundTripper) http.RoundTripper {
	return &Decorator{
		Decorator: DecorateHeaders(map[string]string{
			"Authorization": "Bearer " + token,
		}),
		Base: base,
	}
}

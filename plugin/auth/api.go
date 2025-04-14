package auth

import "net/http"

type Authorizer interface {
	Transport(base http.RoundTripper) (http.RoundTripper, error)
}

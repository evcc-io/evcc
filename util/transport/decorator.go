package transport

import (
	"net/http"
)

// Decorator is an http.RoundTripper that makes HTTP requests,
// wrapping a base RoundTripper and modifying given base requests.
type Decorator struct {
	// Decorator modifies the outgoing request
	Decorator func(*http.Request) error

	// Base is the base RoundTripper used to make HTTP requests.
	Base http.RoundTripper
}

// RoundTrip decorates the request using the Decorator.
func (t *Decorator) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Decorator == nil {
		return t.Base.RoundTrip(req)
	}

	reqBodyClosed := false
	if req.Body != nil {
		defer func() {
			if !reqBodyClosed {
				req.Body.Close()
			}
		}()
	}

	req2 := cloneRequest(req) // per RoundTripper contract
	if err := t.Decorator(req2); err != nil {
		return nil, err
	}

	// req.Body is assumed to be closed by the base RoundTripper.
	reqBodyClosed = true
	return t.Base.RoundTrip(req2)
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	r2.Header = r.Header.Clone()
	return r2
}

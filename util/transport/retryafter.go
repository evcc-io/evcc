package transport

import (
	"net/http"
	"strconv"
	"time"
)

type retryAfter struct {
	max  time.Duration
	base http.RoundTripper
}

// RetryAfter creates an http transport that waits out the Retry-After cooldown of a 429
// response and repeats the request, if the cooldown is at most max. Waiting counts
// towards the client timeout.
func RetryAfter(max time.Duration, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}

	return &retryAfter{max: max, base: base}
}

func (t *retryAfter) RoundTrip(req *http.Request) (*http.Response, error) {
	for {
		resp, err := t.base.RoundTrip(req)
		if err != nil || resp.StatusCode != http.StatusTooManyRequests {
			return resp, err
		}

		sec, err := strconv.Atoi(resp.Header.Get("Retry-After"))
		d := time.Duration(sec) * time.Second
		if err != nil || d < 0 || d > t.max || (req.Body != nil && req.Body != http.NoBody && req.GetBody == nil) {
			return resp, nil
		}
		resp.Body.Close()

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(d):
		}

		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req = req.Clone(req.Context())
			req.Body = body
		}
	}
}

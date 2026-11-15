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

// cooldown parses Retry-After as delay seconds or HTTP date and reports if it is at most max
func (t *retryAfter) cooldown(v string) (time.Duration, bool) {
	var d time.Duration
	if sec, err := strconv.Atoi(v); err == nil {
		if sec < 0 || sec > int(t.max/time.Second) {
			return 0, false
		}
		d = time.Duration(sec) * time.Second
	} else if ts, err := http.ParseTime(v); err == nil {
		d = time.Until(ts)
	} else {
		return 0, false
	}

	// zero or past cooldowns would retry in a burst
	return max(d, time.Second), d <= t.max
}

func (t *retryAfter) RoundTrip(req *http.Request) (*http.Response, error) {
	for {
		resp, err := t.base.RoundTrip(req)
		if err != nil || resp.StatusCode != http.StatusTooManyRequests {
			return resp, err
		}

		d, ok := t.cooldown(resp.Header.Get("Retry-After"))
		if !ok || (req.Body != nil && req.Body != http.NoBody && req.GetBody == nil) {
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

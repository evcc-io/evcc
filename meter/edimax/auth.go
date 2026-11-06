package edimax

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/evcc-io/evcc/util/transport"
)

// authTransport picks basic (V1) or digest (V2/V3) auth from the plug's challenge,
// so basic credentials are never sent to digest devices
type authTransport struct {
	basic, digest http.RoundTripper
	isBasic       atomic.Bool
}

func newAuthTransport(user, password string, base http.RoundTripper) http.RoundTripper {
	return &authTransport{
		basic:  transport.BasicAuth(user, password, base),
		digest: transport.Digest(user, password, base),
	}
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.isBasic.Load() {
		return t.basic.RoundTrip(req)
	}

	// the digest transport probes without credentials and answers digest challenges itself
	resp, err := t.digest.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized || !basicChallenge(resp.Header) {
		return resp, err
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	t.isBasic.Store(true)

	retry := req.Clone(req.Context())
	if req.GetBody != nil {
		if retry.Body, err = req.GetBody(); err != nil {
			return nil, err
		}
	}

	return t.basic.RoundTrip(retry)
}

func basicChallenge(h http.Header) bool {
	for _, v := range h.Values("WWW-Authenticate") {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(v)), "basic") {
			return true
		}
	}
	return false
}

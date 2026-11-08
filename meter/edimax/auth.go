package edimax

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/evcc-io/evcc/util/transport"
)

const (
	schemeUnknown int32 = iota
	schemeBasic
	schemeDigest
)

// authTransport picks basic (V1) or digest (V2/V3) auth from the plug's first challenge
// and pins it, so basic credentials are never sent once a plug answered via digest
type authTransport struct {
	basic, digest http.RoundTripper
	scheme        atomic.Int32
}

func newAuthTransport(user, password string, base http.RoundTripper) http.RoundTripper {
	return &authTransport{
		basic:  transport.BasicAuth(user, password, base),
		digest: transport.Digest(user, password, base),
	}
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch t.scheme.Load() {
	case schemeBasic:
		return t.basic.RoundTrip(req)
	case schemeDigest:
		return t.digest.RoundTrip(req)
	}

	// the digest transport probes without credentials and answers digest challenges itself
	resp, err := t.digest.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.scheme.CompareAndSwap(schemeUnknown, schemeDigest)
		return resp, nil
	}

	if !basicChallenge(resp.Header) || !t.scheme.CompareAndSwap(schemeUnknown, schemeBasic) {
		return resp, nil
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

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

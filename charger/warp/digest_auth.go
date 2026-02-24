package warp

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type DigestTransport struct {
	Username  string
	Password  string
	Base      http.RoundTripper
	mu        sync.Mutex
	challenge *DigestChallenge
}

type DigestChallenge struct {
	Realm     string
	Nonce     string
	Qop       string
	Opaque    string
	Algorithm string
}

func ParseDigestChallenge(h string) (*DigestChallenge, error) {
	dc := &DigestChallenge{}
	parts := strings.Split(h, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "Digest ") {
			p = strings.TrimPrefix(p, "Digest ")
		}
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.Trim(kv[1], `"`)
		switch key {
		case "realm":
			dc.Realm = val
		case "nonce":
			dc.Nonce = val
		case "opaque":
			dc.Opaque = val
		case "qop":
			dc.Qop = val
		case "algorithm":
			dc.Algorithm = val
		}
	}
	return dc, nil
}

func BuildDigestAuthHeader(ch *DigestChallenge, method, uri, user, pass string) string {
	ha1 := md5Hex(fmt.Sprintf("%s:%s:%s", user, ch.Realm, pass))
	ha2 := md5Hex(fmt.Sprintf("%s:%s", method, uri))

	cnonce := randomHex(16)
	nc := "00000001"

	response := md5Hex(fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		ha1, ch.Nonce, nc, cnonce, ch.Qop, ha2))

	return fmt.Sprintf(
		`Digest username="%s", realm="%s", nonce="%s", uri="%s", algorithm="MD5", response="%s", qop=%s, nc=%s, cnonce="%s"`,
		user, ch.Realm, ch.Nonce, uri, response, ch.Qop, nc, cnonce,
	)
}

func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (t *DigestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Base == nil {
		t.Base = http.DefaultTransport
	}
	// If we already have a challenge → send digest directly
	t.mu.Lock()
	ch := t.challenge
	t.mu.Unlock()
	if ch != nil {
		return t.roundTripWithDigest(req, ch)
	}
	// 1. First try without Auth
	resp, err := t.Base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	// 2. Parse challenge
	hdr := resp.Header.Get("WWW-Authenticate")
	ch, err = ParseDigestChallenge(hdr)
	if err != nil {
		return resp, err
	}
	// Save challenge
	t.mu.Lock()
	t.challenge = ch
	t.mu.Unlock()
	// Reread body (if necessery)
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
	}
	// 3. Second try with Digest Auth
	return t.roundTripWithDigest(req, ch)
}

func (t *DigestTransport) roundTripWithDigest(req *http.Request, ch *DigestChallenge) (*http.Response, error) {
	// Copy request
	r2 := req.Clone(req.Context())
	uri := r2.URL.RequestURI()
	auth := BuildDigestAuthHeader(ch, r2.Method, uri, t.Username, t.Password)
	r2.Header.Set("Authorization", auth)
	return t.Base.RoundTrip(r2)
}

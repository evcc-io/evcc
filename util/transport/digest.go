package transport

import (
	"errors"
	"net/http"
	"strings"

	"github.com/icholy/digest"
)

// nonRfcSha256 is sent by some servers instead of the RFC 7616 compliant SHA-256
const nonRfcSha256 = "SHA256"

// Digest creates an http transport performing digest auth. The challenge is
// cached per host, so all requests after the first authenticate preemptively
// instead of paying a 401 challenge round trip each time (RFC 7616 §3.3).
func Digest(user, password string, base http.RoundTripper) http.RoundTripper {
	return &digest.Transport{
		Username:      user,
		Password:      password,
		Transport:     base,
		FindChallenge: digestChallenge,
		Digest:        digestCredentials,
	}
}

// digestChallenge additionally accepts challenges announcing the non-RFC SHA256
func digestChallenge(h http.Header) (*digest.Challenge, error) {
	chal, err := digest.FindChallenge(h)
	if err == nil || !errors.Is(err, digest.ErrNoChallenge) {
		return chal, err
	}

	for _, header := range h.Values("WWW-Authenticate") {
		if !digest.IsDigest(header) {
			continue
		}
		if chal, err := digest.ParseChallenge(header); err == nil && strings.EqualFold(chal.Algorithm, nonRfcSha256) {
			return chal, nil
		}
	}

	return nil, err
}

// digestCredentials hashes a non-RFC SHA256 challenge as SHA-256, echoing back
// the spelling the server used
func digestCredentials(_ *http.Request, chal *digest.Challenge, opt digest.Options) (*digest.Credentials, error) {
	if !strings.EqualFold(chal.Algorithm, nonRfcSha256) {
		return digest.Digest(chal, opt)
	}

	rfc := *chal
	rfc.Algorithm = "SHA-256"

	cred, err := digest.Digest(&rfc, opt)
	if err != nil {
		return nil, err
	}
	cred.Algorithm = chal.Algorithm

	return cred, nil
}

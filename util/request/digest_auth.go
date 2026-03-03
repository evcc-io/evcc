package request

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	"github.com/icholy/digest"
)

// interner Cache: key = origin|realm (hier vereinfachend origin)
var (
    digMu   sync.Mutex
    digChal = map[string]*digest.Challenge{}
)

func originKey(u *url.URL) string {
    return u.Scheme + "://" + u.Host
}


func setCachedChallenge(u *url.URL, chal *digest.Challenge) {
    digMu.Lock()
    digChal[originKey(u)] = chal
    digMu.Unlock()
}
func getCachedChallenge(u *url.URL) *digest.Challenge {
    digMu.Lock()
    defer digMu.Unlock()
    return digChal[originKey(u)]
}

func extractChallenge(h http.Header, find ...func(http.Header) (*digest.Challenge, error)) (*digest.Challenge, error) {
    // Default: WWW-Authenticate
    if len(find) == 0 {
        val := h.Get("WWW-Authenticate")
        if val == "" || !digest.IsDigest(val) {
            return nil, digest.ErrNoChallenge
        }
        return digest.ParseChallenge(val)
    }
    // User function to extract the correct header
    return find[0](h)
}


// PrimeDigest fetches and caches the digest challenge for the given URL.
func (r *Helper) PrimeDigest(urlStr string) error {
    u, err := url.Parse(urlStr)
    if err != nil {
        return err
    }

	switch u.Scheme {
	case "wss":
		u.Scheme = "https"
	case "ws":
		u.Scheme = "http"
	default:
		return fmt.Errorf("Unknown protocol scheme: %s", u.Scheme)
	}

	tc, _ := r.getDigestTransportCreds()
	var finder func(http.Header) (*digest.Challenge, error)
	if tc != nil && tc.findChallenge != nil {
		finder = tc.findChallenge
	}

	// HEAD first (some servers only challengen on the exact path)
	req, _ := http.NewRequest(http.MethodHead, u.String(), nil)
	resp, err := r.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_,_ = io.Copy(io.Discard, resp.Body)

	chal, err := extractChallenge(resp.Header)
	if err == nil {
		setCachedChallenge(u, chal)
		return nil
	}

    // Fallback: GET without Auth
    req2, _ := http.NewRequest(http.MethodGet, u.String(), nil)
    resp2, err := r.Do(req2)
    if err != nil {
        return err
    }
    defer resp2.Body.Close()
    _, _ = io.Copy(io.Discard, resp2.Body)

    chal, err = extractChallenge(resp2.Header, finder)
    if err != nil {
        return err
    }
    digMu.Lock()
    digChal[originKey(u)] = chal
    digMu.Unlock()
    return nil
}

// DigestHeader builds an Authorization header using a cached challenge.
func (r *Helper) DigestHeader(method, urlStr string, body []byte, explicitUserPass...string) (http.Header, error) {
    u, err := url.Parse(urlStr)
    if err != nil {
        return nil, err
    }

	switch u.Scheme {
	case "wss":
		u.Scheme = "https"
	case "ws":
		u.Scheme = "http"
	default:
		return nil, fmt.Errorf("Unknown protocol scheme: %s", u.Scheme)
	}

	chal := getCachedChallenge(u)
	if chal == nil {
		return nil, fmt.Errorf("no cached digest challenge; call PrimeDigest first")
	}

	var username, password string
    // 1) Prefer creds from transport if available
    if tc, err2 := r.getDigestTransportCreds(); err2 == nil && (tc.username != "" || tc.password != "") {
        username, password = tc.username, tc.password
    } else if len(explicitUserPass) == 2 {
        // 2) fallback to explicitly provided credentials
        username, password = explicitUserPass[0], explicitUserPass[1]
    } else if err2 == nil && tc != nil && tc.digestFn != nil {
        // 3) as a last resort: ask the transport's Digest callback to generate credentials directly
        req, _ := http.NewRequest(method, u.String(), nil)
        opt := digest.Options{
            Method: method,
            URI:    u.RequestURI(),
        }
        // support qop=auth-int if body is present
        if len(body) > 0 {
            buf := bytes.NewBuffer(append([]byte(nil), body...))
            opt.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(buf.Bytes())), nil }
        }
        cred, err := tc.digestFn(req, chal, opt)
        if err != nil {
            return nil, err
        }
        h := make(http.Header)
        h.Set("Authorization", cred.String())
        return h, nil
    } else {
        return nil, fmt.Errorf("no credentials found on transport; provide username/password")
    }

    // Low-level path using username/password from above
    opt := digest.Options{
        Username: username,
        Password: password,
        Method:   method,
        URI:      u.RequestURI(),
        Count:    1, // falls du dieselbe Challenge mehrfach verwendest: Count inkrementieren
    }
    if len(body) > 0 {
        b := append([]byte(nil), body...)
        opt.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(b)), nil }
    }

    cred, err := digest.Digest(chal, opt)
    if err != nil {
        return nil, err
    }
    h := make(http.Header)
    h.Set("Authorization", cred.String())
    return h, nil
}


// tries to locate *digest.Transport inside a possibly wrapped RoundTripper chain
func findDigestTransport(rt http.RoundTripper) (*digest.Transport, bool) {
    // direct hit
    if dt, ok := rt.(*digest.Transport); ok {
        return dt, true
    }
    // reflectively walk common "wrapper" patterns that embed a "Transport http.RoundTripper" field
    // (euer NewTripper/transport.Default() nutzt genau so ein Muster)
    v := reflect.ValueOf(rt)
    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }
    if v.Kind() != reflect.Struct {
        return nil, false
    }
    // try field named "Transport"
    f := v.FieldByName("Transport")
    if f.IsValid() && f.CanInterface() {
        if inner, ok := f.Interface().(http.RoundTripper); ok {
            return findDigestTransport(inner)
        }
    }
    return nil, false
}

type transportCreds struct {
    username      string
    password      string
    findChallenge func(http.Header) (*digest.Challenge, error)
    digestFn      func(req *http.Request, chal *digest.Challenge, opt digest.Options) (*digest.Credentials, error)
}

func (r *Helper) getDigestTransportCreds() (*transportCreds, error) {
    if r == nil || r.Client == nil || r.Transport == nil {
        return nil, fmt.Errorf("no transport available on helper")
    }
    dt, ok := findDigestTransport(r.Transport)
    if !ok || dt == nil {
        return nil, fmt.Errorf("digest transport not found")
    }
    tc := &transportCreds{
        username:      dt.Username,
        password:      dt.Password,
        findChallenge: dt.FindChallenge,
        digestFn:      dt.Digest,
    }
    return tc, nil
}

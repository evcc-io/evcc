package ghostone

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

type tokenSource struct {
	*request.Helper
	mu       sync.Mutex
	ts       oauth2.TokenSource
	log      *util.Logger
	uri      string
	user     string
	password string
}

// Transport creates an http transport authenticating with a JWT token against
// the ghost REST API. A request rejected with 401 is retried once with a new
// token since the wallbox invalidates tokens on restart (#33753).
func Transport(ctx context.Context, log *util.Logger, uri, user, password string, base http.RoundTripper) (http.RoundTripper, error) {
	c := &tokenSource{
		Helper:   request.NewHelper(log),
		log:      log,
		uri:      uri + "/jwt/login",
		user:     user,
		password: password,
	}

	c.Client.Transport = transport.Insecure()

	token, err := c.login(ctx)
	if err != nil {
		return nil, err
	}

	c.ts = oauth.RefreshTokenSource(log, token, c.refresh)

	return &authTransport{
		ts:   c,
		base: &oauth2.Transport{Source: c, Base: base},
	}, nil
}

type authTransport struct {
	ts   *tokenSource
	base http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}

	retry := req.Clone(req.Context())

	if req.Body != nil {
		if req.GetBody == nil {
			return resp, nil
		}

		body, err := req.GetBody()
		if err != nil {
			return resp, nil
		}

		retry.Body = body
	}

	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	t.ts.invalidate()

	return t.base.RoundTrip(retry)
}

func (c *tokenSource) Token() (*oauth2.Token, error) {
	c.mu.Lock()
	ts := c.ts
	c.mu.Unlock()

	return ts.Token()
}

// invalidate drops the current token, forcing a new login on the next request
func (c *tokenSource) invalidate() {
	c.mu.Lock()
	c.ts = oauth.RefreshTokenSource(c.log, nil, c.refresh)
	c.mu.Unlock()
}

func (c *tokenSource) login(ctx context.Context) (*oauth2.Token, error) {
	data := url.Values{
		"user": {c.user},
		"pass": {c.password},
	}

	req, err := request.New(http.MethodPost, c.uri, strings.NewReader(data.Encode()), request.URLEncoding)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := request.ResponseError(resp); err != nil {
		return nil, err
	}

	token := resp.Header.Get("Authorization")
	if token == "" {
		return nil, errors.New("missing authorization header")
	}

	// strip "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")

	// extract expiry from JWT claims
	expiry := time.Now().Add(10 * time.Minute) // fallback
	var claims jwt.RegisteredClaims
	if _, _, err := jwt.NewParser(jwt.WithoutClaimsValidation()).ParseUnverified(token, &claims); err == nil && claims.ExpiresAt != nil {
		expiry = claims.ExpiresAt.Time
	}

	return &oauth2.Token{
		AccessToken: token,
		TokenType:   "Bearer",
		Expiry:      expiry,
	}, nil
}

func (c *tokenSource) refresh(_ *oauth2.Token) (*oauth2.Token, error) {
	// re-login to get new token (no context needed- refresh runs in steady-state, not during init)
	return c.login(context.Background())
}

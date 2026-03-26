package ghostone

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

type tokenSource struct {
	*request.Helper
	oauth2.TokenSource
	uri      string
	user     string
	password string
}

// TokenSource creates a JWT token source for the ghost REST API
func TokenSource(log *util.Logger, uri, user, password string) (oauth2.TokenSource, error) {
	c := &tokenSource{
		Helper:   request.NewHelper(log),
		uri:      uri + "/jwt/login",
		user:     user,
		password: password,
	}

	c.Client.Transport = transport.Insecure()

	token, err := c.login()
	if err != nil {
		return nil, err
	}

	c.TokenSource = oauth.RefreshTokenSource(token, c.refresh)

	return c, nil
}

func (c *tokenSource) login() (*oauth2.Token, error) {
	data := url.Values{
		"user": {c.user},
		"pass": {c.password},
	}

	req, err := request.New(http.MethodPost, c.uri, strings.NewReader(data.Encode()), request.URLEncoding)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
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

	return &oauth2.Token{
		AccessToken: token,
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour), // assume 1h validity
	}, nil
}

func (c *tokenSource) refresh(_ *oauth2.Token) (*oauth2.Token, error) {
	// re-login to get new token
	return c.login()
}

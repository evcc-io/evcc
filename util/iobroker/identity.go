package iobroker

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type Identity struct {
	*request.Helper
	user     string
	password string
	uri      string
	oauth2.TokenSource
}

// NewConnection creates a new Iobroker connection
func NewIdentity(log *util.Logger, uri, username, password string) (*Identity, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	if username == "" || password == "" {
		return nil, errors.New("invalid username or password")
	}

	i := &Identity{
		Helper:   request.NewHelper(log),
		user:     username,
		password: password,
		uri:      uri,
	}

	_, err := i.Login()

	return i, err
}

// URI returns the base URI of the iobroker instance
func (c *Identity) URI() string {
	return c.uri
}

// Token returns the base URI of the Home Assistant instance
func (c *Identity) Token() (*oauth2.Token, error) {
	if c.TokenSource == nil {
		return c.Login()
	}
	return c.TokenSource.Token()
}

// Login authenticates with given payload
func (c *Identity) login(params url.Values) (*oauth2.Token, error) {
	uri := fmt.Sprintf("%s/oauth/token", c.uri)

	params.Add("stayloggedin", "true")
	req, err := request.New(http.MethodPost, uri, strings.NewReader(params.Encode()), map[string]string{
		"Content-Type": request.FormContent,
		"Accept":       "application/json",
	})

	var token oauth2.Token
	if err == nil {
		req.SetBasicAuth("ioBroker", "ioBroker")
		err = c.DoJSON(req, &token)
		token.Expiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	return &token, err
}

// Login authenticates with username/password
func (c *Identity) Login() (*oauth2.Token, error) {
	data := url.Values{
		"grant_type": {"password"},
		"username":   {c.user},
		"password":   {c.password},
	}
	token, err := c.login(data)
	if err == nil {
		c.TokenSource = oauth.RefreshTokenSource(token, c.refreshToken)
	}

	return token, err
}

// refreshToken renews the token
func (c *Identity) refreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
	}

	token, err := c.login(data)

	return token, err
}

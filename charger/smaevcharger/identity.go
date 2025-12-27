package smaevcharger

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// Token is the SMAEVCHarger22 Token
// Auth Token Data json Response structure
type Token struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
}

func (t *Token) AsOAuth2Token() *oauth2.Token {
	if t == nil {
		return nil
	}

	return &oauth2.Token{
		AccessToken:  t.AccessToken,
		TokenType:    t.TokenType,
		RefreshToken: t.RefreshToken,
		Expiry:       time.Now().Add(time.Second * time.Duration(t.ExpiresIn)),
	}
}

// tokenSource is an oauth2.TokenSource
type tokenSource struct {
	*request.Helper
	oauth2.TokenSource
	uri      string
	user     string
	password string
}

// TokenSource creates an SMAevCharger token source
func TokenSource(log *util.Logger, uri, user, password string) (oauth2.TokenSource, error) {
	c := &tokenSource{
		Helper:   request.NewHelper(log),
		uri:      uri + "/token",
		user:     user,
		password: password,
	}

	data := url.Values{
		"grant_type": {"password"},
		"password":   {password},
		"username":   {user},
	}

	req, err := request.New(http.MethodPost, c.uri, strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil {
		var token Token
		if err = c.DoJSON(req, &token); err == nil {
			c.TokenSource = oauth.RefreshTokenSource(token.AsOAuth2Token(), c.refreshToken)
		}
	}

	return c, err
}

func (c *tokenSource) refreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
	}

	req, err := request.New(http.MethodPost, c.uri, strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil {
		var token Token
		if err = c.DoJSON(req, &token); err == nil {
			return token.AsOAuth2Token(), nil
		}
	}

	return nil, err
}

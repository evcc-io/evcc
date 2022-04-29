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
	//UiIdleTime    string `json:"uiIdleTime"`
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
	Host     string
	User     string
	Password string
}

// TokenSource creates an Easee token source
func TokenSource(log *util.Logger, host, user, password string) (oauth2.TokenSource, error) {
	c := &tokenSource{
		Helper:   request.NewHelper(log),
		Host:     host,
		User:     user,
		Password: password,
	}

	Uri := c.Host + "/token"
	data := url.Values{
		"grant_type": {"password"},
		"password":   {password},
		"username":   {user},
	}

	req, err := request.New(http.MethodPost, Uri, strings.NewReader(data.Encode()), request.URLEncoding)

	if err == nil {
		var token Token
		if err = c.DoJSON(req, &token); err == nil {
			c.TokenSource = oauth.RefreshTokenSource(token.AsOAuth2Token(), c)
		}
	}

	return c, err
}

func (c *tokenSource) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	/*

		Uri := c.Host + "/refresh_token" //URL Unknown?!
		data := url.Values{
			"AccessToken":		{token.AccessToken},
			"RefreshToken":		{token.RefreshToken},
		}

		req, err := request.New(http.MethodPost, Uri, strings.NewReader(data.Encode()),request.JSONEncoding)

		var res *oauth2.Token
		if err == nil {
			var refreshed Token
			if err = c.DoJSON(req, &refreshed); err == nil {
				res = refreshed.AsOAuth2Token()
			}
		}

		return res, err

	*/

	//Token refresh not working, as a workaround we aquire a new token with the password and the username

	Uri := c.Host + "/token"
	data := url.Values{
		"grant_type": {"password"},
		"password":   {c.Password},
		"username":   {c.User},
	}

	req, err := request.New(http.MethodPost, Uri, strings.NewReader(data.Encode()), request.URLEncoding)

	if err == nil {
		var token Token
		if err = c.DoJSON(req, &token); err == nil {
			return token.AsOAuth2Token(), nil
		}
	}
	return nil, err
}

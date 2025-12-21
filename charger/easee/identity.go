package easee

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// Token is the Easee Token
type Token struct {
	AccessToken  string  `json:"accessToken"`
	ExpiresIn    float32 `json:"expiresIn"`
	TokenType    string  `json:"tokenType"`
	RefreshToken string  `json:"refreshToken"`
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
	user, password string
}

// TokenSourceCache stores per-user token sources
var TokenSourceCache = oauth.NewTokenSourceCache()

// GetTokenSource returns a shared oauth2.TokenSource for the given user credentials.
// Multiple chargers using the same user credentials will share the same TokenSource,
// ensuring tokens are reused and authentication is deduplicated.
func GetTokenSource(log *util.Logger, user, password string) (oauth2.TokenSource, error) {
	// Check if token source exists in cache
	if ts, exists := TokenSourceCache.Get(user, password); exists {
		return ts, nil
	}

	c := &tokenSource{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}

	token, err := c.authenticate()
	if err != nil {
		return nil, err
	}

	ts := oauth.RefreshTokenSource(token.AsOAuth2Token(), c)
	TokenSourceCache.Set(user, password, ts)

	return ts, nil
}

func (c *tokenSource) authenticate() (*Token, error) {
	data := struct {
		Username string `json:"userName"`
		Password string `json:"password"`
	}{
		Username: c.user,
		Password: c.password,
	}

	uri := fmt.Sprintf("%s/%s", API, "accounts/login")
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return nil, err
	}

	var token Token
	err = c.DoJSON(req, &token)

	return &token, err
}

func (c *tokenSource) RefreshToken(oauthToken *oauth2.Token) (*oauth2.Token, error) {
	data := struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}{
		AccessToken:  oauthToken.AccessToken,
		RefreshToken: oauthToken.RefreshToken,
	}

	uri := fmt.Sprintf("%s/%s", API, "accounts/refresh_token")
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return nil, err
	}

	var token *Token
	if err := c.DoJSON(req, &token); err != nil {
		// re-login
		if token, err = c.authenticate(); err != nil {
			return nil, err
		}
	}

	return token.AsOAuth2Token(), nil
}

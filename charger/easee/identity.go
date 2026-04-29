package easee

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/cache"
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

// tokenSource is an oauth2.TokenSource and holds Easee account credentials and performs authentication.
type tokenSource struct {
	*request.Helper
	user, password string
}

// NewIdentity creates an Identity for the given credentials.
func NewIdentity(log *util.Logger, user, password string) *tokenSource {
	return &tokenSource{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}
}

// tokenSourceCache stores per-user token sources
var tokenSourceCache = cache.New[oauth2.TokenSource]()

// TokenSource returns a shared oauth2.TokenSource for the given user.
func TokenSource(log *util.Logger, user, password string) (oauth2.TokenSource, error) {
	return tokenSourceCache.GetOrCreate(user, func() (oauth2.TokenSource, error) {
		id := NewIdentity(log, user, password)
		token, err := id.Authenticate()
		if err != nil {
			return nil, err
		}
		return oauth.RefreshTokenSource(token, id.RefreshToken), nil
	})
}

// TokenSourceWithInitial creates a token source, using initialToken if non-nil,
// or performing a fresh login otherwise. Unlike TokenSource, this does not cache.
func (c *tokenSource) TokenSourceWithInitial(initialToken *oauth2.Token) (oauth2.TokenSource, error) {
	if initialToken == nil {
		token, err := c.Authenticate()
		if err != nil {
			return nil, err
		}
		initialToken = token
	}
	return oauth.RefreshTokenSource(initialToken, c.RefreshToken), nil
}

// Authenticate performs the initial username/password login and returns an oauth2.Token.
func (c *tokenSource) Authenticate() (*oauth2.Token, error) {
	token, err := c.login()
	if err != nil {
		return nil, err
	}
	return token.AsOAuth2Token(), nil
}

func (c *tokenSource) login() (*Token, error) {
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

// RefreshToken refreshes an existing oauth2 token using the Easee refresh endpoint.
// Falls back to a full re-login when the refresh endpoint rejects the token.
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
		// re-login on refresh failure
		if token, err = c.login(); err != nil {
			return nil, err
		}
	}

	return token.AsOAuth2Token(), nil
}

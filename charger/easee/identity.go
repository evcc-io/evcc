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
}

// TokenSource creates an Easee token source
func TokenSource(log *util.Logger, user, password string) (oauth2.TokenSource, error) {
	c := &tokenSource{
		Helper: request.NewHelper(log),
	}

	data := struct {
		Username string `json:"userName"`
		Password string `json:"password"`
	}{
		Username: user,
		Password: password,
	}

	uri := fmt.Sprintf("%s/%s", API, "accounts/login")
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return nil, err
	}

	var token Token
	if err := c.DoJSON(req, &token); err != nil {
		return nil, err
	}

	oauthToken := token.AsOAuth2Token()
	ts := oauth2.ReuseTokenSourceWithExpiry(oauthToken, oauth.RefreshTokenSource(oauthToken, c), 15*time.Minute)

	return ts, nil
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
		return oauthToken, err
	}

	var token Token
	if err := c.DoJSON(req, &token); err != nil {
		return oauthToken, err
	}

	return token.AsOAuth2Token(), nil
}

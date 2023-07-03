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
	oauth2.TokenSource
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

	if err == nil {
		var token Token
		if err = c.DoJSON(req, &token); err == nil {
			token := token.AsOAuth2Token()
			ts := oauth.RefreshTokenSource(token, c)
			c.TokenSource = oauth2.ReuseTokenSourceWithExpiry(token, ts, 6*time.Hour)
		}
	}

	return c, err
}

func (c *tokenSource) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
	}

	uri := fmt.Sprintf("%s/%s", API, "accounts/refresh_token")
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	var res *oauth2.Token
	if err == nil {
		var refreshed Token
		if err = c.DoJSON(req, &refreshed); err == nil {
			res = refreshed.AsOAuth2Token()
		}
	}

	return res, err
}

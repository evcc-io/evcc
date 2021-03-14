package id

import (
	"net/http"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

// Token is the non-OIDC compliant VW ID token structure
type Token struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	Expiry       time.Time
}

func (t *Token) Expire(d time.Duration) {
	t.Expiry = time.Now().Add(d)
}

func NewTokenSource(log *util.Logger, token Token) oauth2.TokenSource {
	return &TokenSource{
		token:  token,
		Helper: request.NewHelper(log),
	}
}

type TokenSource struct {
	token Token
	*request.Helper
}

func (ts *TokenSource) Token() (*oauth2.Token, error) {
	var err error
	if time.Until(ts.token.Expiry) < time.Minute {
		err = ts.refreshToken()
	}

	ot := &oauth2.Token{
		AccessToken:  ts.token.AccessToken,
		RefreshToken: ts.token.RefreshToken,
		Expiry:       ts.token.Expiry,
	}

	return ot, err
}

func (ts *TokenSource) refreshToken() error {
	uri := "https://login.apps.emea.vwapps.io/refresh/v1"

	headers := map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + ts.token.RefreshToken,
	}

	req, err := request.New(http.MethodGet, uri, nil, headers)
	if err == nil {
		var token Token
		if err = ts.DoJSON(req, &token); err == nil {
			token.Expire(3600 * time.Second)
			ts.token = token
		}
	}

	return err
}

package id

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

// Token is the VW ID token
type Token oauth2.Token

func (t *Token) UnmarshalJSON(data []byte) error {
	var s struct {
		AccessToken  string
		RefreshToken string
	}

	err := json.Unmarshal(data, &s)
	if err == nil {
		t.TokenType = "bearer"
		t.AccessToken = s.AccessToken
		t.RefreshToken = s.RefreshToken
		t.Expiry = time.Now().Add(time.Hour)
	}

	return err
}

func (t *Token) TokenSource(log *util.Logger) oauth2.TokenSource {
	return &TokenSource{
		Helper: request.NewHelper(log),
		token:  t,
	}
}

type TokenSource struct {
	*request.Helper
	token *Token
}

func (ts *TokenSource) Token() (*oauth2.Token, error) {
	var err error
	if time.Until(ts.token.Expiry) < time.Minute {
		err = ts.refreshToken()
	}

	return (*oauth2.Token)(ts.token), err
}

func (ts *TokenSource) refreshToken() error {
	uri := "https://login.apps.emea.vwapps.io/refresh/v1"

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + ts.token.RefreshToken,
	})

	if err == nil {
		var token Token
		if err = ts.DoJSON(req, &token); err == nil {
			ts.token = &token
		}
	}

	return err
}

package vw

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/oidc"
	"golang.org/x/oauth2"
)

// Token is the VW token
type Token oidc.Token

func (t *Token) TokenSource(log *util.Logger, clientID string) oauth2.TokenSource {
	return &TokenSource{
		Helper:   request.NewHelper(log),
		clientID: clientID,
		token:    t,
	}
}

type TokenSource struct {
	*request.Helper
	clientID string
	token    *Token
}

func (ts *TokenSource) Token() (*oauth2.Token, error) {
	var err error
	if time.Until(ts.token.Expiry) < time.Minute {
		err = ts.refreshToken()
	}

	return &ts.token.Token, err
}

func (ts *TokenSource) refreshToken() error {
	data := url.Values(map[string][]string{
		"grant_type":    {"refresh_token"},
		"refresh_token": {ts.token.RefreshToken},
		"scope":         {"sc2:fal"},
	})

	req, err := request.New(http.MethodPost, OauthTokenURI, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"X-Client-Id":  ts.clientID,
	})

	if err == nil {
		var token Token
		if err = ts.DoJSON(req, &token); err == nil {
			ts.token = &token
		}
	}

	return err
}

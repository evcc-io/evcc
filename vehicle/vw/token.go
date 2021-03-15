package vw

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/oidc"
	"github.com/imdario/mergo"
	"golang.org/x/oauth2"
)

// Token is the VW token
type Token oauth2.Token

func (t *Token) UnmarshalJSON(data []byte) error {
	var o oidc.Token

	err := json.Unmarshal(data, &o)
	if err == nil {
		*t = (Token)(o.Token)
	}

	return err
}

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

	return (*oauth2.Token)(ts.token), err
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
			ts.mergeToken(token)
		}
	}

	return err
}

func (ts *TokenSource) mergeToken(t Token) {
	if err := mergo.Merge(ts.token, &t, mergo.WithOverride); err != nil {
		panic(err)
	}
}

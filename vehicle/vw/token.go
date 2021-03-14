package vw

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

// Token is the VW token
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

func (t *Token) UnmarshalJSON(data []byte) error {
	var s struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
		Scope        string `json:"scope,omitempty"`
		ExpiresIn    int64  `json:"expires_in,omitempty"`
	}

	err := json.Unmarshal(data, &s)
	if err == nil {
		t.AccessToken = s.AccessToken
		t.RefreshToken = s.RefreshToken
		t.Expiry = time.Now().Add(time.Second * time.Duration(s.ExpiresIn))
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

	ot := &oauth2.Token{
		AccessToken:  ts.token.AccessToken,
		RefreshToken: ts.token.RefreshToken,
		Expiry:       ts.token.Expiry,
	}

	return ot, err
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

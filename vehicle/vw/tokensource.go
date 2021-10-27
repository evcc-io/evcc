package vw

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type TokenSource struct {
	*request.Helper
	identity PlatformLogin
	clientID string
	query    url.Values
	user     string
	password string
}

var _ TokenSourceProvider = (*TokenSource)(nil)

func NewTokenSource(log *util.Logger, identity PlatformLogin, clientID string, query url.Values, user, password string) *TokenSource {
	return &TokenSource{
		Helper:   request.NewHelper(log),
		identity: identity,
		clientID: clientID,
		query:    query,
		user:     user,
		password: password,
	}
}

func (v *TokenSource) TokenSource() (oauth2.TokenSource, error) {
	token, err := v.login()
	if err != nil {
		return nil, err
	}

	return oauth.RefreshTokenSource(&token.Token, v), nil
}

// LoginVAG performs VAG login and finally exchanges id token for access and refresh tokens
func (v *TokenSource) login() (Token, error) {
	var token Token
	uri := fmt.Sprintf("%s/oidc/v1/authorize?%s", IdentityURI, v.query.Encode())

	q, err := v.identity.UserLogin(uri, v.user, v.password)

	if err == nil && q.Get("id_token") == "" {
		err = errors.New("missing id_token")
	}

	if err == nil {
		data := url.Values(map[string][]string{
			"grant_type": {"id_token"},
			"scope":      {"sc2:fal"},
			"token":      {q.Get("id_token")},
		})

		var req *http.Request
		req, err = request.New(http.MethodPost, OauthTokenURI, strings.NewReader(data.Encode()), map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"X-Client-Id":  v.clientID,
		})

		if err == nil {
			err = v.DoJSON(req, &token)
		}

		// check if token response contained error
		if errT := token.Error(); err != nil && errT != nil {
			err = fmt.Errorf("token exchange: %w", errT)
		}
	}

	return token, err
}

// RefreshToken implements oauth.TokenRefresher
func (v *TokenSource) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values(map[string][]string{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
		"scope":         {"sc2:fal"},
	})

	req, err := request.New(http.MethodPost, OauthTokenURI, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"X-Client-Id":  v.clientID,
	})

	var res Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	if se, ok := err.(request.StatusError); ok && se.HasStatus(http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden) {
		res, err = v.login()
	}

	return &res.Token, err
}

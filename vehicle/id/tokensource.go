package id

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vw"
	"golang.org/x/oauth2"
)

const (
	// OauthTokenURI is the login uri for ID vehicles
	OauthTokenURI = "https://login.apps.emea.vwapps.io"
)

type TokenSource struct {
	*request.Helper
	identity vw.PlatformLogin
	query    url.Values
	user     string
	password string
}

var _ vw.TokenSourceProvider = (*TokenSource)(nil)

func NewTokenSource(log *util.Logger, identity vw.PlatformLogin, query url.Values, user, password string) *TokenSource {
	return &TokenSource{
		Helper:   request.NewHelper(log),
		identity: identity,
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

	return oauth.RefreshTokenSource((*oauth2.Token)(&token), v), nil
}

func (v *TokenSource) login() (Token, error) {
	var token Token
	uri := fmt.Sprintf("%s/authorize?%s", OauthTokenURI, v.query.Encode())

	q, err := v.identity.UserLogin(uri, v.user, v.password)

	if err == nil {
		for _, k := range []string{"state", "id_token", "access_token", "code"} {
			if !q.Has(k) {
				err = errors.New("missing " + k)
				break
			}
		}
	}

	if err == nil {
		data := map[string]string{
			"state":             q.Get("state"),
			"id_token":          q.Get("id_token"),
			"redirect_uri":      "weconnect://authenticated",
			"region":            "emea",
			"access_token":      q.Get("access_token"),
			"authorizationCode": q.Get("code"),
		}

		var req *http.Request
		uri = fmt.Sprintf("%s/login/v1", OauthTokenURI)
		req, err = request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

		if err == nil {
			err = v.DoJSON(req, &token)
		}
	}

	return token, err
}

func (v *TokenSource) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	uri := fmt.Sprintf("%s/refresh/v1", OauthTokenURI)

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + token.RefreshToken,
	})

	var res Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	if se, ok := err.(request.StatusError); ok && se.HasStatus(http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden) {
		res, err = v.login()
	}

	return (*oauth2.Token)(&res), err
}

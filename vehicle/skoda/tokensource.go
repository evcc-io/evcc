package skoda

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vw"
	"golang.org/x/oauth2"
)

const (
	// OauthTokenURI is the token service uri for Skoda Enyaq vehicles
	OauthTokenURI = "https://tokenrefreshservice.apps.emea.vwapps.io"
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

	return oauth.RefreshTokenSource(&token.Token, v), nil
}

func (v *TokenSource) login() (vw.Token, error) {
	var token vw.Token
	uri := fmt.Sprintf("%s/oidc/v1/authorize?%s", vw.IdentityURI, v.query.Encode())

	q, err := v.identity.UserLogin(uri, v.user, v.password)

	if err == nil {
		for _, k := range []string{"id_token", "code"} {
			if !q.Has(k) {
				err = errors.New("missing " + k)
				break
			}
		}
	}

	if err == nil {
		data := url.Values(map[string][]string{
			"auth_code": {q.Get("code")},
			"id_token":  {q.Get("id_token")},
			"brand":     {"skoda"},
		})

		var req *http.Request
		uri = fmt.Sprintf("%s/exchangeAuthCode", OauthTokenURI)
		req, err = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)

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
	uri := fmt.Sprintf("%s/refreshTokens", OauthTokenURI)

	data := url.Values(map[string][]string{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
		"brand":         {"skoda"},
	})

	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)

	var res vw.Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	if se, ok := err.(request.StatusError); ok && se.HasStatus(http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden) {
		res, err = v.login()
	}

	return &res.Token, err
}

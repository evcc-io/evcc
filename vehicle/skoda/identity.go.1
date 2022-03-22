package skoda

import (
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

type Identity struct {
	*request.Helper
	idtp *vw.IDTokenProvider
	oauth2.TokenSource
}

func NewIdentity(log *util.Logger, query url.Values, user, password string) *Identity {
	uri := fmt.Sprintf("%s/oidc/v1/authorize?%s", vw.IdentityURI, query.Encode())

	return &Identity{
		Helper: request.NewHelper(log),
		idtp:   vw.NewIDTokenProvider(log, uri, user, password),
	}
}

func (v *Identity) Login() error {
	token, err := v.login()
	if err != nil {
		return err
	}

	v.TokenSource = oauth.RefreshTokenSource(&token.Token, v)

	return nil
}

func (v *Identity) login() (vw.Token, error) {
	q, err := v.idtp.Login()

	if err == nil {
		err = util.RequireValues(q, "id_token", "code")
	}

	var token vw.Token
	if err == nil {
		data := url.Values(map[string][]string{
			"auth_code": {q.Get("code")},
			"id_token":  {q.Get("id_token")},
			"brand":     {"skoda"},
		})

		var req *http.Request
		uri := fmt.Sprintf("%s/exchangeAuthCode", OauthTokenURI)
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
func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
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

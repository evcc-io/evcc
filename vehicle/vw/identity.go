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

// https://identity.vwgroup.io/.well-known/openid-configuration

const (
	// IdentityURI is the VW OIDC identity provider uri
	IdentityURI = "https://identity.vwgroup.io"

	// OauthTokenURI is used for refreshing tokens
	OauthTokenURI = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/oauth2/v1/token"

	// OauthRevokeURI is used for revoking tokens
	OauthRevokeURI = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/oauth2/v1/revoke"
)

type Identity struct {
	*request.Helper
	oauth2.TokenSource
	idtp      *IDTokenProvider
	clientID  string
	refresher oauth.TokenRefresher
}

func NewIdentity(log *util.Logger, clientID string, query url.Values, user, password string) *Identity {
	uri := fmt.Sprintf("%s/oidc/v1/authorize?%s", IdentityURI, query.Encode())

	return &Identity{
		Helper:    request.NewHelper(log),
		clientID:  clientID,
		idtp:      NewIDTokenProvider(log, uri, user, password),
		refresher: NewTokenRefresher(log, clientID),
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

// login performs the login using the brand-specific subflow for obtaining the
// id token and finally exchanges the id token for access and refresh tokens
func (v *Identity) login() (Token, error) {
	q, err := v.idtp.Login()

	if err == nil && q.Get("id_token") == "" {
		err = errors.New("missing id_token")
	}

	var token Token
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
func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	token, err := v.refresher.RefreshToken(token)
	if err == nil {
		return token, nil
	}

	// re-login
	var res Token
	if se, ok := err.(request.StatusError); ok && se.HasStatus(http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden) {
		res, err = v.login()
	}

	return &res.Token, err
}

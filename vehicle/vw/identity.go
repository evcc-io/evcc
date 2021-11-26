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
	idtp       *IDTokenProvider
	clientID   string
	storageKey string
	oauth2.TokenSource
}

func NewIdentity(log *util.Logger, clientID string, query url.Values, user, password string) *Identity {
	uri := fmt.Sprintf("%s/oidc/v1/authorize?%s", IdentityURI, query.Encode())

	return &Identity{
		Helper:     request.NewHelper(log),
		idtp:       NewIDTokenProvider(log, uri, user, password),
		clientID:   clientID,
		storageKey: HashString(fmt.Sprintf("%s%s%s", uri, user, password)),
	}
}

func (v *Identity) Login() error {
	if token := Restore(v.storageKey); token != nil {
		v.TokenSource = oauth.RefreshTokenSource(token, v)
		return nil
	}

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

	if err == nil {
		Persist(v.storageKey, &token.Token)
	}

	return token, err
}

// RefreshToken implements oauth.TokenRefresher
func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
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

	if err == nil {
		Persist(v.storageKey, &res.Token)
	}

	return &res.Token, err
}

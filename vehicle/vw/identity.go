package vw

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
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
	idtp     *IDTokenProvider
	clientID string
	store    string
	oauth2.TokenSource
}

func NewIdentity(log *util.Logger, clientID string, query url.Values, user, password string) *Identity {
	uri := fmt.Sprintf("%s/oidc/v1/authorize?%s", IdentityURI, query.Encode())
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s%s%s", uri, user, password)))

	return &Identity{
		Helper:   request.NewHelper(log),
		idtp:     NewIDTokenProvider(log, uri, user, password),
		clientID: clientID,
		store:    hex.EncodeToString(hash[:]) + ".token",
	}
}

func (v *Identity) Login() error {
	if r, err := os.Open(v.store); err == nil {
		var token oauth2.Token
		if err := gob.NewDecoder(r).Decode(&token); err == nil && token.Valid() {
			v.TokenSource = oauth.RefreshTokenSource(&token, v)
			return nil
		}
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
		v.persistToken(&token.Token)
	}

	return token, err
}

func (v *Identity) persistToken(token *oauth2.Token) {
	if w, err := os.OpenFile(v.store, os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		_ = gob.NewEncoder(w).Encode(token)
	}
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
		v.persistToken(&res.Token)
	}

	return &res.Token, err
}

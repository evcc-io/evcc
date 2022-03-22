package vw

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// TokenRefresher performs the MBBOauth token refresh without re-login
type TokenRefresher struct {
	*request.Helper
	clientID string
}

// NewTokenRefresher creates an MBBOauth token refresher
func NewTokenRefresher(log *util.Logger, clientID string) *TokenRefresher {
	return &TokenRefresher{
		Helper:   request.NewHelper(log),
		clientID: clientID,
	}
}

// RefreshToken implements oauth.TokenRefresher
func (v *TokenRefresher) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
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

	return &res.Token, err
}

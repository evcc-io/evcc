package weconnect

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"github.com/evcc-io/evcc/vehicle/vag/cariad"
	"golang.org/x/oauth2"
)

const tokenURL = cariad.BaseURL + "/auth/v1/idk/oidc/token"

// ExchangeCode swaps the authorization code from the OIDC callback for an
// access/refresh token pair at the cariad BFF OIDC token endpoint. Replaces
// the legacy WeConnect SSO exchange (/user-login/login/v1), which VW removed
// in May 2026.
func ExchangeCode(log *util.Logger, q url.Values) (*oauth2.Token, error) {
	if err := urlvalues.Require(q, "code", "code_verifier"); err != nil {
		return nil, err
	}

	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {q.Get("code")},
		"code_verifier": {q.Get("code_verifier")},
		"redirect_uri":  {"weconnect://authenticated"},
		"client_id":     {cariad.ClientID},
	}

	return postToken(log, data)
}

// TokenSource returns a refreshing oauth2.TokenSource that swaps refresh
// tokens for fresh access tokens at the same OIDC token endpoint.
func TokenSource(log *util.Logger, token *oauth2.Token) oauth2.TokenSource {
	return oauth.RefreshTokenSource(token, func(t *oauth2.Token) (*oauth2.Token, error) {
		data := url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {t.RefreshToken},
			"client_id":     {cariad.ClientID},
		}
		return postToken(log, data)
	})
}

func postToken(log *util.Logger, data url.Values) (*oauth2.Token, error) {
	req, err := request.New(http.MethodPost, tokenURL, strings.NewReader(data.Encode()),
		map[string]string{
			"Content-Type":           request.FormContent,
			"Accept":                 request.JSONContent,
			"User-Agent":             cariad.UserAgent,
			"x-android-package-name": cariad.AndroidPackageName,
		})
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	if err := request.NewHelper(log).DoJSON(req, &token); err != nil {
		return nil, err
	}

	token.Expiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &token, nil
}

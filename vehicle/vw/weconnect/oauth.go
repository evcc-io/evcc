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

// tokenValidity is how long an access token is treated as valid. VW's IDK token
// endpoint no longer issues usable refresh tokens (the app uses the OIDC hybrid
// flow, which never returns a refresh_token); the access token expires after
// ~2 hours, after which a full username/password re-login is required.
const tokenValidity = 2 * time.Hour

// ExchangeCode swaps the authorization code from the OIDC callback for an
// access token at the cariad BFF OIDC token endpoint. Replaces the legacy
// WeConnect SSO exchange (/user-login/login/v1), which VW removed in May 2026.
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

// TokenSource returns an oauth2.TokenSource that performs a full re-login via
// the login function once the access token expires. VW no longer issues usable
// refresh tokens, so refreshing is replaced by re-running the username/password
// login.
func TokenSource(token *oauth2.Token, login func() (*oauth2.Token, error)) oauth2.TokenSource {
	return oauth.RefreshTokenSource(token, func(*oauth2.Token) (*oauth2.Token, error) {
		return login()
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

	// VW no longer issues usable refresh tokens; treat the access token as
	// expiring after a fixed validity, after which the caller re-logs in.
	token.Expiry = time.Now().Add(tokenValidity)
	return &token, nil
}

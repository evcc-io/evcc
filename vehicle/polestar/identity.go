package polestar

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.polestar

// OAuth endpoints and credentials
const (
	OAuthURI    = "https://polestarid.eu.polestar.com"
	ClientID    = "l3oopkc_10"
	RedirectURI = "https://www.polestar.com/sign-in-callback"
)

type Identity struct {
	*request.Helper
	user, password string
	log            *util.Logger
	token          *oauth2.Token
}

// NewIdentity creates Polestar identity
func NewIdentity(log *util.Logger, user, password string) (*Identity, error) {
	v := &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
		log:      log,
	}

	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, err
	}
	v.Client.Jar = jar

	// Get initial token
	token, err := v.login()
	if err != nil {
		return nil, err
	}
	v.token = token

	return v, nil
}

func (v *Identity) login() (*oauth2.Token, error) {
	cv := oauth2.GenerateVerifier()

	data := url.Values{
		"client_id":             {ClientID},
		"redirect_uri":          {RedirectURI},
		"response_type":         {"code"},
		"state":                 {lo.RandomString(16, lo.AlphanumericCharset)},
		"scope":                 {"openid", "profile", "email"},
		"code_challenge":        {oauth2.S256ChallengeFromVerifier(cv)},
		"code_challenge_method": {"S256"},
	}

	// Request authorization URL with browser-like headers
	uri := fmt.Sprintf("%s/as/authorization.oauth2?%s", OAuthURI, data.Encode())
	req, _ := request.New(http.MethodGet, uri, nil, map[string]string{
		"Accept": "text/html,application/xhtml+xml,application/xml;",
	})

	resp, err := v.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Extract resume path from HTML response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	htmlContent := string(body)
	re := regexp.MustCompile(`url: "(/as/[^/]+)/resume/as/authorization\.ping"`)
	matches := re.FindStringSubmatch(htmlContent)

	if len(matches) < 2 {
		return nil, errors.New("could not find resume path")
	}

	resumePath := strings.TrimPrefix(matches[1], "/as/")

	// Submit credentials to login endpoint
	loginURL := fmt.Sprintf("%s/as/%s/resume/as/authorization.ping", OAuthURI, resumePath)
	data = url.Values{
		"pf.username": {v.user},
		"pf.pass":     {v.password},
		"client_id":   {ClientID},
	}

	req, _ = request.New(http.MethodPost, loginURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Accept":       "application/json",
	})

	resp, err = v.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Extract authorization code from response
	code := resp.Request.URL.Query().Get("code")
	if code == "" {
		return nil, errors.New("missing authorization code")
	}

	// Exchange code for token
	data = url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {cv},
		"client_id":     {ClientID},
		"redirect_uri":  {RedirectURI},
	}

	var token oauth2.Token
	req, _ = request.New(http.MethodPost, OAuthURI+"/as/token.oauth2",
		strings.NewReader(data.Encode()),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"Accept":       "application/json",
		},
	)

	if err := v.DoJSON(req, &token); err != nil {
		return nil, err
	}

	return util.TokenWithExpiry(&token), nil
}

// TokenSource implements oauth.TokenSource
func (v *Identity) TokenSource() oauth2.TokenSource {
	return oauth2.ReuseTokenSource(v.token, v)
}

// Token implements oauth.TokenSource
func (v *Identity) Token() (*oauth2.Token, error) {
	if v.token == nil || !v.token.Valid() {
		token, err := v.login()
		if err != nil {
			return nil, err
		}
		v.token = token
	}
	return v.token, nil
}

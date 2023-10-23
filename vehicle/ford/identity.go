package ford

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

const (
	AuthURI       = "https://sso.ci.ford.com"
	TokenURI      = "https://api.mps.ford.com"
	ClientID      = "9fb503e0-715b-47e8-adfd-ad4b7770f73b"
	ApplicationID = "1E8C7794-FF5F-49BC-9596-A1E0C86C5B19"
)

var OAuth2Config = &oauth2.Config{
	ClientID: ClientID,
	Endpoint: oauth2.Endpoint{
		AuthURL:  fmt.Sprintf("%s/v1.0/endpoint/default/authorize", AuthURI),
		TokenURL: fmt.Sprintf("%s/oidc/endpoint/default/token", AuthURI),
	},
	RedirectURL: "fordapp://userauthorized",
	Scopes:      []string{"openid"},
}

type Identity struct {
	*request.Helper
	user, password string
	oauth2.TokenSource
}

// NewIdentity creates Ford identity
func NewIdentity(log *util.Logger, user, password string) *Identity {
	return &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}
}

// Login authenticates with username/password to get new aws credentials
func (v *Identity) Login() error {
	token, err := v.login()
	if err == nil {
		v.TokenSource = oauth.RefreshTokenSource((*oauth2.Token)(token), v)
	}
	return err
}

// login authenticates with username/password to get new token
func (v *Identity) login() (*oauth.Token, error) {
	cv := oauth2.GenerateVerifier()

	state := lo.RandomString(16, lo.AlphanumericCharset)
	uri := OAuth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(cv),
		oauth2.SetAuthURLParam("max_age", "3600"),
	)

	v.Jar, _ = cookiejar.New(nil)
	defer func() { v.Jar = nil }()

	var body []byte
	req, err := request.New(http.MethodGet, uri, nil, nil)
	if err == nil {
		body, err = v.DoBody(req)
	}

	if err != nil {
		return nil, err
	}

	match := regexp.MustCompile(`data-ibm-login-url="(.+?)"`).FindSubmatch(body)
	if len(match) < 2 {
		return nil, errors.New("missing login url")
	}

	data := url.Values{
		"operation":       {"verify"},
		"login-form-type": {"pwd"},
		"username":        {v.user},
		"password":        {v.password},
	}

	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return http.ErrUseLastResponse
		}
		return nil
	}
	defer func() { v.Client.CheckRedirect = nil }()

	uri = fmt.Sprintf("%s%s", AuthURI, string(match[1]))
	req, err = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)

	var loc *url.URL
	if err == nil {
		var resp *http.Response
		if resp, err = v.Do(req); err == nil {
			defer resp.Body.Close()

			if body, err = io.ReadAll(resp.Body); err == nil {
				if match := regexp.MustCompile(`data-ibm-login-error-text="(.+?)"`).FindSubmatch(body); len(match) >= 2 {
					err = errors.New(string(match[1]))
				}
			}
		}

		if err == nil {
			loc, err = url.Parse(resp.Header.Get("location"))
		}
	}

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), oauth2.HTTPClient, v.Client),
		request.Timeout,
	)
	defer cancel()

	code := loc.Query().Get("code")
	if code == "" {
		return nil, errors.New("could not obtain auth code- check user and password")
	}

	tok, err := OAuth2Config.Exchange(ctx, code, oauth2.VerifierOption(cv))

	// exchange code for api token
	var token oauth.Token
	if err == nil {
		data := map[string]string{
			"ciToken": tok.AccessToken,
		}

		uri := fmt.Sprintf("%s/api/token/v2/cat-with-ci-access-token", TokenURI)

		var req *http.Request
		req, err = request.New(http.MethodPost, uri, request.MarshalJSON(data), map[string]string{
			"Content-type":   request.JSONContent,
			"Application-Id": ApplicationID,
		})

		if err == nil {
			err = v.DoJSON(req, &token)
		}
	}

	return &token, err
}

// RefreshToken implements oauth.TokenRefresher
func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := map[string]string{
		"refresh_token": token.RefreshToken,
	}

	uri := fmt.Sprintf("%s/api/token/v2/cat-with-refresh-token", TokenURI)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), map[string]string{
		"Content-type":   request.JSONContent,
		"Application-Id": ApplicationID,
	})

	var res *oauth.Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	if err != nil {
		res, err = v.login()
	}

	return (*oauth2.Token)(res), err
}

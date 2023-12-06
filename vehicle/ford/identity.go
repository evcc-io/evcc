package ford

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	TokenURI      = "https://api.mps.ford.com"
	LoginUri      = "https://login.ford.com/4566605f-43a7-400a-946e-89cc9fdb0bd7/B2C_1A_SignInSignUp_de-DE"
	ClientID      = "09852200-05fd-41f6-8c21-d36d3497dc64"
	ApplicationID = "1E8C7794-FF5F-49BC-9596-A1E0C86C5B19"
)

var loginHeaders = map[string]string{
	"Accept":          "*/*",
	"Accept-Language": "en-us",
	"User-Agent":      "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1",
}

var OAuth2Config = &oauth2.Config{
	ClientID: ClientID,
	Endpoint: oauth2.Endpoint{
		AuthURL:  fmt.Sprintf("%s/oauth2/v2.0/authorize", LoginUri),
		TokenURL: fmt.Sprintf("%s/oauth2/v2.0/token", LoginUri),
	},
	RedirectURL: "fordapp://userauthorized",
	Scopes:      []string{ClientID, "openid"},
}

type Settings struct {
	TransId string `json:"transId"`
	Csrf    string `json:"csrf"`
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
		oauth2.SetAuthURLParam("ui_locales", "de-DE"),
		oauth2.SetAuthURLParam("language_code", "de-DE"),
		oauth2.SetAuthURLParam("country_code", "DE"),
		oauth2.SetAuthURLParam("ford_application_id", ApplicationID),
	)

	v.Jar, _ = cookiejar.New(nil)
	defer func() { v.Jar = nil }()

	var body []byte
	req, err := request.New(http.MethodGet, uri, nil, loginHeaders)
	if err == nil {
		body, err = v.DoBody(req)
	}

	if err != nil {
		return nil, err
	}

	match := regexp.MustCompile(`var SETTINGS = (\{[^;]*\});`).FindSubmatch(body)
	if len(match) < 2 {
		return nil, errors.New("missing settings variable")
	}

	var settings Settings

	if err := json.Unmarshal([]byte(match[1]), &settings); err != nil {
		return nil, err
	}

	data := url.Values{
		"request_type": {"RESPONSE"},
		"signInName":   {v.user},
		"password":     {v.password},
	}

	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return http.ErrUseLastResponse
		}
		return nil
	}
	defer func() { v.Client.CheckRedirect = nil }()

	uri2 := fmt.Sprintf("%s/SelfAsserted?tx=%s&p=B2C_1A_SignInSignUp_de-DE", LoginUri, settings.TransId)
	req, err = request.New(http.MethodPost, uri2, strings.NewReader(data.Encode()), request.URLEncoding, map[string]string{
		"Accept":          "*/*",
		"Accept-Language": "en-us",
		"User-Agent":      "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1",
		"Origin":          "https://login.ford.com",
		"Referer":         uri,
		"X-Csrf-Token":    settings.Csrf,
	})

	if err == nil {
		var resp *http.Response
		if resp, err = v.Do(req); err == nil {
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				return nil, errors.New("self-assert failed")
			}
		}
	}

	uri3 := fmt.Sprintf("%s/api/CombinedSigninAndSignup/confirmed?rememberMe=false&csrf_token=%s&tx=%s&p=B2C_1A_SignInSignUp_de-DE", LoginUri, settings.Csrf, settings.TransId)
	req, err = request.New(http.MethodGet, uri3, nil, request.URLEncoding, map[string]string{
		"Origin":       "https://login.ford.com",
		"Referer":      uri,
		"X-Csrf-Token": settings.Csrf,
	})

	var loc *url.URL
	if err == nil {
		var resp *http.Response
		if resp, err = v.Do(req); err == nil {
			defer resp.Body.Close()
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
			"idpToken": tok.AccessToken,
		}

		uri := fmt.Sprintf("%s/api/token/v2/cat-with-b2c-access-token", TokenURI)

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

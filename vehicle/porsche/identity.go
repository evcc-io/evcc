package porsche

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

const (
	OAuthURI = "https://login.porsche.com"
)

// https://login.porsche.com/.well-known/openid-configuration
var (
	OAuth2Config = &oauth2.Config{
		ClientID:    "4mPO3OE5Srjb1iaUGWsbqKBvvesya8oA",
		RedirectURL: "https://my.porsche.com/core/de/de_DE/",
		Endpoint: oauth2.Endpoint{
			AuthURL:  OAuthURI + "/as/authorization.oauth2",
			TokenURL: OAuthURI + "/as/token.oauth2",
		},
		Scopes: []string{"openid"},
	}

	EmobilityOAuth2Config = &oauth2.Config{
		ClientID:    "NJOxLv4QQNrpZnYQbb7mCvdiMxQWkHDq",
		RedirectURL: "https://my.porsche.com/myservices/auth/auth.html",
		Endpoint:    OAuth2Config.Endpoint,
		Scopes:      []string{"openid"},
	}
)

// Identity is the Porsche Identity client
type Identity struct {
	*request.Helper
	user, password                 string
	defaultToken, emobilityToken   *oauth2.Token
	DefaultSource, EmobilitySource oauth2.TokenSource
}

// NewIdentity creates Porsche identity
func NewIdentity(log *util.Logger, user, password string) *Identity {
	v := &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}

	return v
}

func (v *Identity) Login() error {
	_, err := v.RefreshToken(nil)

	if err == nil {
		v.DefaultSource = oauth.RefreshTokenSource(v.defaultToken, v)
		v.EmobilitySource = oauth.RefreshTokenSource(v.emobilityToken, &emobilityAdapter{v})
	}

	return err
}

// RefreshToken performs new login and creates default and emobility tokens
func (v *Identity) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, err
	}

	// track cookies and follow all (>10) redirects
	v.Client.Jar = jar
	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return nil
	}
	defer func() {
		v.Client.Jar = nil
		v.Client.CheckRedirect = nil
	}()

	preLogin := url.Values{
		"sec":          []string{""},
		"resume":       []string{""},
		"thirdPartyId": []string{""},
		"state":        []string{""},
		"username":     []string{v.user},
		"password":     []string{v.password},
		"keeploggedin": []string{"false"},
	}

	// get the login page
	uri := fmt.Sprintf("%s/auth/api/v1/de/de_DE/public/login", OAuthURI)
	resp, err := v.PostForm(uri, preLogin)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	query, err := url.ParseQuery(resp.Request.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	dataLoginAuth := url.Values{
		"sec":          []string{query.Get("sec")},
		"resume":       []string{query.Get("resume")},
		"thirdPartyId": []string{query.Get("thirdPartyID")},
		"state":        []string{query.Get("state")},
		"username":     []string{v.user},
		"password":     []string{v.password},
		"keeploggedin": []string{"false"},
	}

	// process the auth so the session is authenticated
	resp, err = v.PostForm(uri, dataLoginAuth)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	// get the token for the generic API
	token, err := v.fetchToken(OAuth2Config)
	if err == nil {
		v.defaultToken = token

		if token, err = v.fetchToken(EmobilityOAuth2Config); err == nil {
			v.emobilityToken = token
		}
	}

	return v.defaultToken, err
}

func (v *Identity) fetchToken(oc *oauth2.Config) (*oauth2.Token, error) {
	cv, err := cv.CreateCodeVerifier()
	if err != nil {
		return nil, err
	}

	uri := oc.AuthCodeURL("uvobn7XJs1", oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", cv.CodeChallengeS256()),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("country", "de"),
		oauth2.SetAuthURLParam("locale", "de_DE"),
	)

	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return http.ErrUseLastResponse
		}
		return nil
	}

	resp, err := v.Client.Get(uri)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	query, err := url.ParseQuery(resp.Request.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	code := query.Get("code")
	if code == "" {
		return nil, errors.New("no auth code")
	}

	ctx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), oauth2.HTTPClient, v.Client),
		request.Timeout,
	)
	defer cancel()

	token, err := oc.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", cv.CodeChallengePlain()),
	)

	return token, err
}

type emobilityAdapter struct {
	tr *Identity
}

func (v *emobilityAdapter) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	token, err := v.tr.RefreshToken(nil)
	if err == nil {
		token = v.tr.emobilityToken
	}
	return token, err
}

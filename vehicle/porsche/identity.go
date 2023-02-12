package porsche

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

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

	MobileOAuth2Config = &oauth2.Config{
		ClientID:    "L20OiZ0kBgWt958NWbuCB8gb970y6V6U",
		RedirectURL: "One-Product-App://porsche-id/oauth2redirect",
		Endpoint:    OAuth2Config.Endpoint,
		Scopes:      []string{"openid", "magiclink", "mbb"},
	}
)

// Identity is the Porsche Identity client
type Identity struct {
	*request.Helper
	user, password                               string
	defaultToken, emobilityToken, mobileToken    *oauth2.Token
	DefaultSource, EmobilitySource, MobileSource oauth2.TokenSource
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
		v.MobileSource = oauth.RefreshTokenSource(v.mobileToken, &mobileAdapter{v})
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

	// get the login page
	uri := fmt.Sprintf("%s/auth/api/v1/de/de_DE/public/login", OAuthURI)
	resp, err := v.Get(uri)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	query, err := url.ParseQuery(resp.Request.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	sec := query.Get("sec")
	resume := query.Get("resume")
	state := query.Get("state")
	thirdPartyID := query.Get("thirdPartyId")

	dataLoginAuth := url.Values{
		"sec":          []string{sec},
		"resume":       []string{resume},
		"thirdPartyId": []string{thirdPartyID},
		"state":        []string{state},
		"username":     []string{v.user},
		"password":     []string{v.password},
		"keeploggedin": []string{"false"},
	}

	req, err := request.New(http.MethodPost, uri, strings.NewReader(dataLoginAuth.Encode()), request.URLEncoding)
	if err != nil {
		return nil, err
	}

	// process the auth so the session is authenticated
	if resp, err = v.Client.Do(req); err != nil {
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

		if token, err = v.fetchToken(MobileOAuth2Config); err == nil {
			v.mobileToken = token
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

	rawQuery := resp.Request.URL.RawQuery
	if location, ok := strings.CutPrefix(resp.Header.Get("Location"), "One-Product-App://porsche-id/oauth2redirect?"); ok {
		rawQuery = location
	}
	query, err := url.ParseQuery(rawQuery)
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

type mobileAdapter struct {
	tr *Identity
}

func (v *mobileAdapter) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	token, err := v.tr.RefreshToken(nil)
	if err == nil {
		token = v.tr.mobileToken
	}
	return token, err
}

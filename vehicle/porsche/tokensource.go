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
	"github.com/evcc-io/evcc/util/request"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

type tokenRefresher struct {
	*request.Helper
	user, password               string
	DefaultToken, EmobilityToken *oauth2.Token
}

func newTokenRefresher(log *util.Logger, user, password string) *tokenRefresher {
	return &tokenRefresher{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}
}

func (v *tokenRefresher) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
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
		v.DefaultToken = token

		if token, err = v.fetchToken(EmobilityOAuth2Config); err == nil {
			v.EmobilityToken = token
		}
	}

	return v.DefaultToken, err
}

func (v *tokenRefresher) fetchToken(oc *oauth2.Config) (*oauth2.Token, error) {
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
	tr *tokenRefresher
}

func (v *emobilityAdapter) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	token, err := v.tr.RefreshToken(nil)
	if err == nil {
		token = v.tr.EmobilityToken
	}
	return token, err
}

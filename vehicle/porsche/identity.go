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
		Scopes:      OAuth2Config.Scopes,
	}
)

// Identity is the Porsche Identity client
type Identity struct {
	log *util.Logger
	*request.Helper
	DefaultSource, EmobilitySource oauth2.TokenSource
}

// NewIdentity creates Porsche identity
func NewIdentity(log *util.Logger) *Identity {
	v := &Identity{
		log:    log,
		Helper: request.NewHelper(log),
	}

	return v
}

func (v *Identity) Login(user, password string) error {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return err
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
		return err
	}
	resp.Body.Close()

	query, err := url.ParseQuery(resp.Request.URL.RawQuery)
	if err != nil {
		return err
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
		"username":     []string{user},
		"password":     []string{password},
		"keeploggedin": []string{"false"},
	}

	req, err := request.New(http.MethodPost, uri, strings.NewReader(dataLoginAuth.Encode()), request.URLEncoding)
	if err != nil {
		return err
	}

	// process the auth so the session is authenticated
	if resp, err = v.Client.Do(req); err != nil {
		return err
	}
	resp.Body.Close()

	// get the token for the generic API
	token, err := v.fetchToken(OAuth2Config)
	if err == nil {
		v.DefaultSource = OAuth2Config.TokenSource(context.Background(), token)

		if token, err = v.fetchToken(EmobilityOAuth2Config); err == nil {
			v.EmobilitySource = EmobilityOAuth2Config.TokenSource(context.Background(), token)
		}
	}

	return err
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

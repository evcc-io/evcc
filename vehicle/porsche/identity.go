package porsche

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

const (
	OAuthURI = "https://identity.porsche.com"
)

// https://identity.porsche.com/.well-known/openid-configuration
var (
	OAuth2Config = &oauth2.Config{
		ClientID:    "UYsK00My6bCqJdbQhTQ0PbWmcSdIAMig",
		RedirectURL: "https://my.porsche.com/",
		Endpoint: oauth2.Endpoint{
			AuthURL:  OAuthURI + "/authorize",
			TokenURL: OAuthURI + "/oauth/token",
		},
		Scopes: []string{"openid", "offline_access"},
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
	cv, err := cv.CreateCodeVerifier()
	if err != nil {
		return nil, err
	}

	state := lo.RandomString(16, lo.AlphanumericCharset)
	uri := OAuth2Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("audience", ApiURI),
		oauth2.SetAuthURLParam("ui_locales", "de-DE"),
		oauth2.SetAuthURLParam("code_challenge", cv.CodeChallengeS256()),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	v.Client.Jar, _ = cookiejar.New(nil)
	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	defer func() {
		v.Client.Jar = nil
		v.Client.CheckRedirect = nil
	}()

	resp, err := v.Client.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	u, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}

	query := u.Query()
	for _, p := range []string{"client_id", "code_challenge", "scope", "protocol"} {
		query.Del(p)
	}
	for k, v := range map[string]string{
		"connection": "Username-Password-Authentication",
		"tenant":     "porsche-production",
		"sec":        "high",
	} {
		query.Set(k, v)
	}
	query.Set("client_id", OAuth2Config.ClientID)
	query.Set("username", v.user)
	query.Set("password", v.password)

	uri = fmt.Sprintf("%s/usernamepassword/login", OAuthURI)
	resp, err = v.PostForm(uri, query)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	query = make(url.Values)
	doc.Find("input[type=hidden]").Each(func(_ int, el *goquery.Selection) {
		if name, ok := el.Attr("name"); ok {
			val, _ := el.Attr("value")
			query.Set(name, val)
		}
	})

	var code string
	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if code = req.URL.Query().Get("code"); code != "" {
			return http.ErrUseLastResponse
		}
		return nil
	}

	uri = fmt.Sprintf("%s/login/callback", OAuthURI)
	resp, err = v.PostForm(uri, query)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	if code == "" {
		return nil, errors.New("auth code not found")
	}

	ctx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), oauth2.HTTPClient, v.Client),
		request.Timeout,
	)
	defer cancel()

	v.defaultToken, err = OAuth2Config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("client_id", OAuth2Config.ClientID),
		oauth2.SetAuthURLParam("code_verifier", cv.CodeChallengePlain()),
	)

	return v.defaultToken, err
}

// func (v *Identity) fetchToken(oc *oauth2.Config) (*oauth2.Token, error) {
// 	cv, err := cv.CreateCodeVerifier()
// 	if err != nil {
// 		return nil, err
// 	}

// 	uri := oc.AuthCodeURL("uvobn7XJs1", oauth2.AccessTypeOffline,
// 		oauth2.SetAuthURLParam("code_challenge", cv.CodeChallengeS256()),
// 		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
// 		oauth2.SetAuthURLParam("country", "de"),
// 		oauth2.SetAuthURLParam("locale", "de_DE"),
// 	)

// 	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
// 		if req.URL.Scheme != "https" {
// 			return http.ErrUseLastResponse
// 		}
// 		return nil
// 	}

// 	resp, err := v.Client.Get(uri)
// 	if err != nil {
// 		return nil, err
// 	}
// 	resp.Body.Close()

// 	query, err := url.ParseQuery(resp.Request.URL.RawQuery)
// 	if err != nil {
// 		return nil, err
// 	}

// 	code := query.Get("code")
// 	if code == "" {
// 		return nil, errors.New("no auth code")
// 	}

// 	ctx, cancel := context.WithTimeout(
// 		context.WithValue(context.Background(), oauth2.HTTPClient, v.Client),
// 		request.Timeout,
// 	)
// 	defer cancel()

// 	token, err := oc.Exchange(ctx, code,
// 		oauth2.SetAuthURLParam("code_verifier", cv.CodeChallengePlain()),
// 	)

// 	return token, err
// }

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

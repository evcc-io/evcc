package porsche

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

const (
	OAuthURI = "https://identity.porsche.com"
	ClientID = "UYsK00My6bCqJdbQhTQ0PbWmcSdIAMig"
)

// https://identity.porsche.com/.well-known/openid-configuration
var (
	OAuth2Config = &oauth2.Config{
		ClientID:    ClientID,
		RedirectURL: "https://my.porsche.com/",
		Endpoint: oauth2.Endpoint{
			AuthURL:   OAuthURI + "/authorize",
			TokenURL:  OAuthURI + "/oauth/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		Scopes: []string{"openid", "offline_access"},
	}
)

// Identity is the Porsche Identity client
type Identity struct {
	*request.Helper
	user, password string
}

// NewIdentity creates Porsche identity
func NewIdentity(log *util.Logger, user, password string) (oauth2.TokenSource, error) {
	v := &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}

	token, err := v.login()

	return oauth.RefreshTokenSource(token, v), err
}

func (v *Identity) login() (*oauth2.Token, error) {
	cv := oauth2.GenerateVerifier()

	state := lo.RandomString(16, lo.AlphanumericCharset)
	uri := OAuth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(cv),
		oauth2.SetAuthURLParam("audience", ApiURI),
		oauth2.SetAuthURLParam("ui_locales", "de-DE"),
	)

	v.Client.Jar, _ = cookiejar.New(nil)
	v.Client.CheckRedirect = request.DontFollow
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var res struct {
			Description string `json:"description"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err == nil && res.Description != "" {
			return nil, errors.New(res.Description)
		}
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

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

	var param request.InterceptResult
	v.Client.CheckRedirect, param = request.InterceptRedirect("code", true)

	uri = fmt.Sprintf("%s/login/callback", OAuthURI)
	resp, err = v.PostForm(uri, query)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	code, err := param()
	if err != nil {
		return nil, err
	}

	cctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)
	ctx, cancel := context.WithTimeout(cctx, request.Timeout)
	defer cancel()

	return OAuth2Config.Exchange(ctx, code, oauth2.VerifierOption(cv))
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)
	ts := oauth2.ReuseTokenSource(token, OAuth2Config.TokenSource(ctx, token))

	token, err := ts.Token()
	if err != nil {
		token, err = v.login()
	}

	return token, err
}

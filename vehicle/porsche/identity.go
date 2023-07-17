package porsche

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

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
	ClientID = "UYsK00My6bCqJdbQhTQ0PbWmcSdIAMig"

	maxTokenLifetime = time.Hour
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
	log *util.Logger
	*request.Helper
}

// NewIdentity creates Porsche identity
func NewIdentity(log *util.Logger) *Identity {
	v := &Identity{
		log:    log,
		Helper: request.NewHelper(log),
	}

	return v
}

func (v *Identity) Login(oc *oauth2.Config, user, password string) (oauth2.TokenSource, error) {
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
	query.Set("client_id", oc.ClientID)
	query.Set("username", user)
	query.Set("password", password)

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

	token, err := oc.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", cv.CodeChallengePlain()),
	)
	if err != nil {
		return nil, err
	}

	ts := oauth2.ReuseTokenSourceWithExpiry(token, oc.TokenSource(cctx, token), 15*time.Minute)
	go oauth.Refresh(v.log, token, ts, maxTokenLifetime)

	return ts, err
}

package nissan

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

// MyNISSAN OneID (WSO2) authentication constants
const (
	ClientID    = "ZM3WK7ax1OtQKYQ8Qqzcv5VgiA8a"
	Scope       = "openid name profile email offline_access"
	AuthBaseURL = "https://login.mynissan-account.com"
	RedirectURI = "com://wso2.service.nci"
	AuthBrand   = "Nissan"
	AuthClient  = "mynissanapp"

	// the login is not region specific, the locale only selects the form's language
	Locale = "en_GB"

	// Kamereon token exchange constants
	KamereonScope    = "openid profile vehicles"
	KamereonPlatform = "Android"
)

// OAuth2Config is the MyNISSAN OneID authorization code configuration
var OAuth2Config = &oauth2.Config{
	ClientID:    ClientID,
	RedirectURL: RedirectURI,
	Endpoint: oauth2.Endpoint{
		AuthURL:   AuthBaseURL + "/oauth2/authorize",
		TokenURL:  AuthBaseURL + "/oauth2/token",
		AuthStyle: oauth2.AuthStyleInParams,
	},
	Scopes: strings.Split(Scope, " "),
}

var (
	authBaseURI = lo.Must(url.Parse(AuthBaseURL))
	redirectURI = lo.Must(url.Parse(RedirectURI))
)

// ErrInvalidCredentials is returned when Nissan rejects the credentials
var ErrInvalidCredentials = errors.New("invalid credentials")

type Identity struct {
	*request.Helper
	oauth2.TokenSource
	log            *util.Logger
	user, password string
}

// NewIdentity performs the MyNISSAN OneID login and returns a Kamereon token source
func NewIdentity(log *util.Logger, user, password string) (oauth2.TokenSource, error) {
	v := &Identity{
		Helper:   request.NewHelper(log),
		log:      log,
		user:     user,
		password: password,
	}

	token, err := v.login()
	if err != nil {
		return nil, err
	}

	v.TokenSource = oauth.RefreshTokenSource(log, token, v.refresh)

	return v, nil
}

// refresh renews the Kamereon token, falling back to a full login
func (v *Identity) refresh(token *oauth2.Token) (*oauth2.Token, error) {
	if token.RefreshToken != "" {
		res, err := v.refreshKamereonToken(token.RefreshToken)
		if err == nil {
			return res, nil
		}

		v.log.DEBUG.Printf("token refresh failed: %v", err)
	}

	return v.login()
}

// login performs the OneID authorization code flow and exchanges the
// resulting id token for a Kamereon token
func (v *Identity) login() (*oauth2.Token, error) {
	// the login flow is stateful- session and load balancer affinity cookies
	// set while loading the login page must be returned when submitting it
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}

	v.Client.Jar = jar

	// the flow completes by redirecting to the app's custom scheme redirect uri
	// which must be intercepted instead of followed
	var callback *url.URL
	v.Client.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		if isRedirectHost(req.URL) {
			callback = req.URL
			return http.ErrUseLastResponse
		}
		if !isAuthHost(req.URL) {
			return fmt.Errorf("unexpected redirect to %s://%s", req.URL.Scheme, req.URL.Host)
		}
		return nil
	}

	defer func() {
		v.Client.Jar = nil
		v.Client.CheckRedirect = nil
	}()

	cv := oauth2.GenerateVerifier()
	state := lo.RandomString(32, lo.AlphanumericCharset)

	uri := OAuth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(cv),
		oauth2.SetAuthURLParam("locale", Locale),
		oauth2.SetAuthURLParam("brand", AuthBrand),
		oauth2.SetAuthURLParam("client", AuthClient),
	)

	req, err := request.New(http.MethodGet, uri, nil, htmlHeaders)
	if err != nil {
		return nil, err
	}

	resp, err := v.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	form, err := loginForm(resp.Body)
	if err != nil {
		return nil, err
	}

	action, err := resp.Request.URL.Parse(form.Action)
	if err != nil {
		return nil, err
	}

	if !isAuthHost(action) {
		return nil, fmt.Errorf("unexpected login form target %s://%s%s", action.Scheme, action.Host, action.Path)
	}

	// keep the hidden form fields, especially the wso2 sessionDataKey
	data := make(url.Values, len(form.Inputs)+2)
	for k, v := range form.Inputs {
		data.Set(k, v)
	}

	// the user name is region qualified while userName is not
	user := v.user
	if region := form.Inputs["regionCode"]; region != "" {
		user = region + "/" + v.user
	}

	data.Set("userName", v.user)
	data.Set("username", user)
	data.Set("password", v.password)

	headers := map[string]string{
		"Content-Type": request.FormContent,
		"Origin":       action.Scheme + "://" + action.Host,
		"Referer":      resp.Request.URL.String(),
	}
	for k, v := range htmlHeaders {
		headers[k] = v
	}

	if req, err = request.New(http.MethodPost, action.String(), strings.NewReader(data.Encode()), headers); err != nil {
		return nil, err
	}

	resp, err = v.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if callback == nil {
		// Nissan re-renders the login form instead of redirecting when the credentials are rejected
		if _, err := loginForm(resp.Body); err == nil {
			return nil, ErrInvalidCredentials
		}
		return nil, errors.New("login did not complete - open the MyNISSAN app or website, confirm any pending terms, consent or verification prompts, then retry")
	}

	q := callback.Query()
	if err := (&Token{Error: q.Get("error"), ErrorDescription: q.Get("error_description")}).Err(); err != nil {
		return nil, err
	}
	if q.Get("state") != state {
		return nil, errors.New("login state mismatch")
	}

	code := q.Get("code")
	if code == "" {
		return nil, ErrInvalidCredentials
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, v.Client)

	token, err := OAuth2Config.Exchange(ctx, code,
		oauth2.VerifierOption(cv),
		oauth2.SetAuthURLParam("scope", Scope),
	)
	if err != nil {
		return nil, err
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok || idToken == "" {
		return nil, errors.New("missing id token")
	}

	return v.kamereonToken(idToken)
}

// kamereonToken exchanges the OneID id token for a Kamereon token
func (v *Identity) kamereonToken(idToken string) (*oauth2.Token, error) {
	uri := fmt.Sprintf("%s/v1/oauth2/access_token?platform=%s", UserBaseURL, KamereonPlatform)

	req, err := request.New(http.MethodPost, uri, nil, kamereonHeaders(idToken))
	if err != nil {
		return nil, err
	}

	return v.token(req)
}

// refreshKamereonToken renews the Kamereon token using its refresh token
func (v *Identity) refreshKamereonToken(refreshToken string) (*oauth2.Token, error) {
	uri := fmt.Sprintf("%s/v1/oauth2/refresh-token?platform=%s", UserBaseURL, KamereonPlatform)

	data := struct {
		Scope string `json:"scope"`
	}{
		Scope: KamereonScope,
	}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), kamereonHeaders(refreshToken))
	if err != nil {
		return nil, err
	}

	return v.token(req)
}

func (v *Identity) token(req *http.Request) (*oauth2.Token, error) {
	var res Token
	if err := v.DoJSON(req, &res); err != nil {
		// the token endpoints describe the failure in the error response body
		if errT := res.Err(); errT != nil {
			return nil, fmt.Errorf("%w (%w)", errT, err)
		}
		return nil, err
	}

	return res.Token()
}

func kamereonHeaders(authorization string) map[string]string {
	return map[string]string{
		"Authorization": authorization,
		"Content-Type":  "application/vnd.api+json",
		"Accept":        request.JSONContent,
	}
}

var htmlHeaders = map[string]string{
	"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
}

func isAuthHost(u *url.URL) bool {
	return u.Scheme == authBaseURI.Scheme && strings.EqualFold(u.Host, authBaseURI.Host)
}

func isRedirectHost(u *url.URL) bool {
	return u.Scheme == redirectURI.Scheme && strings.EqualFold(u.Host, redirectURI.Host)
}

type formVars struct {
	Action string
	Inputs map[string]string
}

// loginForm extracts the MyNISSAN login form from the given html document
func loginForm(r io.Reader) (formVars, error) {
	var res formVars

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return res, err
	}

	doc.Find("form").EachWithBreak(func(_ int, form *goquery.Selection) bool {
		inputs := make(map[string]string)

		form.Find("input").Each(func(_ int, el *goquery.Selection) {
			if name, ok := el.Attr("name"); ok {
				inputs[name], _ = el.Attr("value")
			}
		})

		// the login form is identified by the wso2 session key and the password field
		if _, ok := inputs["sessionDataKey"]; !ok {
			return true
		}
		if _, ok := inputs["password"]; !ok {
			return true
		}

		res.Action, _ = form.Attr("action")
		res.Inputs = inputs

		return false
	})

	if res.Inputs == nil {
		return res, errors.New("login form not found")
	}
	if res.Action == "" {
		return res, errors.New("login form action not found")
	}

	return res, nil
}

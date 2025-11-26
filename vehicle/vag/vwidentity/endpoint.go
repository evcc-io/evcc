package vwidentity

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"github.com/evcc-io/evcc/vehicle/vag"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"golang.org/x/net/publicsuffix"
)

const (
	BaseURL   = "https://identity.vwgroup.io"
	WellKnown = "https://identity.vwgroup.io/.well-known/openid-configuration"
)

var Config = &oidc.ProviderConfig{
	AuthURL:     "https://identity.vwgroup.io/oidc/v1/authorize",
	TokenURL:    "https://identity.vwgroup.io/oidc/v1/token",
	UserInfoURL: "https://identity-userinfo.vwgroup.io/oidc/userinfo",
}

// Login performs VW identity login with optional code challenge
func Login(log *util.Logger, q url.Values, user, password string) (url.Values, error) {
	return LoginWithAuthURL(log, Config.AuthURL, q, user, password)
}

func LoginWithAuthURL(log *util.Logger, uri string, q url.Values, user, password string) (url.Values, error) {
	var verify func(url.Values)

	// add code challenge
	q = urlvalues.Copy(q)
	if rt := q.Get("response_type"); strings.Contains(rt, "code") {
		verify = vag.ChallengeAndVerifier(q)
	}

	uri = fmt.Sprintf("%s?%s", uri, q.Encode())

	vwi := New(log)
	q, err := vwi.Login(uri, user, password)
	if err != nil {
		return nil, err
	}

	if verify != nil {
		verify(q)
	}

	return q, nil
}

type Service struct {
	*request.Helper
}

func New(log *util.Logger) *Service {
	return &Service{
		Helper: request.NewHelper(log),
	}
}

// Login performs the identity.vwgroup.io login
func (v *Service) Login(uri, user, password string) (url.Values, error) {
	// track cookies and don't follow redirects
	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	v.Client.Jar = jar
	defer func() { v.Client.Jar = nil }()

	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return http.ErrUseLastResponse
		}
		return nil
	}

	// add nonce and state
	query := url.Values{
		"nonce": {lo.RandomString(43, lo.LettersCharset)},
		"state": {uuid.NewString()},
	}

	uri = uri + "&" + query.Encode()

	// GET identity.vwgroup.io/signin-service/v1/signin/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com?relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa
	resp, err := v.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Try to extract legacy form, but don't fail if it's not found
	if vars, err := FormValues(bytes.NewReader(body), "form#emailPasswordForm"); err == nil {
		return v.loginLegacy(vars, user, password)
	}

	return v.loginNew(body, user, password)
}

// loginLegacy performs the legacy VW identity login flow
func (v *Service) loginLegacy(vars FormVars, user, password string) (url.Values, error) {
	var params CredentialParams

	// POST identity.vwgroup.io/signin-service/v1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com/login/identifier
	data := url.Values{
		"_csrf":      {vars.Inputs["_csrf"]},
		"relayState": {vars.Inputs["relayState"]},
		"hmac":       {vars.Inputs["hmac"]},
		"email":      {user},
	}

	uri := BaseURL + vars.Action
	resp, err := v.PostForm(uri, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if params, err = ParseCredentialsPage(resp.Body); err == nil && params.TemplateModel.Error != "" {
		err = errors.New(params.TemplateModel.Error)
	}

	if err != nil {
		return nil, err
	}

	// POST identity.vwgroup.io/signin-service/v1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com/login/authenticate
	data = url.Values{
		"_csrf":      {params.CsrfToken},
		"relayState": {params.TemplateModel.RelayState},
		"hmac":       {params.TemplateModel.Hmac},
		"email":      {user},
		"password":   {password},
	}

	// reuse url from identifier step before
	uri = strings.ReplaceAll(uri, params.TemplateModel.IdentifierUrl, params.TemplateModel.PostAction)

	resp, err = v.PostForm(uri, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, errors.New(resp.Status)
	}

	// GET identity.vwgroup.io/oidc/v1/oauth/sso?clientId=b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com&relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa&userId=bca09cc0-8eba-4110-af71-7242868e1bf1&HMAC=2b01ce6a351fad4dd97dc8110d0967b46c95889ab5010c660a616462e66a83ca
	// GET identity.vwgroup.io/signin-service/v1/consent/users/bca09cc0-8eba-4110-af71-7242868e1bf1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com?scopes=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa&callback=https://identity.vwgroup.io/oidc/v1/oauth/client/callback&hmac=a590931ca3cd9dc3a27f1d1c0c162bf1e5c5c32c9f5b40fcb36d4c6edc631e03
	// GET identity.vwgroup.io/oidc/v1/oauth/client/callback/success?user_id=bca09cc0-8eba-4110-af71-7242868e1bf1&client_id=b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com&scopes=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&consentedScopes=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&relayState=f89a0b750c93e278a7ace170ce374e9cb9eb0a74&hmac=2b728f463c3cfe80f3271fbb35680e5e5218ca70025a46e7fadf7c7982decc2b

	parsed, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}
	return parseAuthLocation(parsed)
}

// loginNew performs the new VW identity login flow
func (v *Service) loginNew(body []byte, user, password string) (url.Values, error) {
	state, err := extractState(body)
	if err != nil {
		return nil, err
	}

	// POST to new login endpoint
	loginData := url.Values{
		"username": {user},
		"password": {password},
		"state":    {state},
	}

	uri := fmt.Sprintf("%s/u/login?state=%s", BaseURL, state)
	resp, err := v.PostForm(uri, loginData)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, errors.New(resp.Status)
	}

	if location == "" {
		return nil, errors.New("no redirect location in new login flow")
	}

	redirectURL, err := resolveLocation(resp.Request.URL, location)
	if err != nil {
		return nil, err
	}

	if redirectURL.Scheme == "https" || redirectURL.Scheme == "http" {
		return nil, fmt.Errorf("unexpected redirect URL scheme: %s, expected app specific url (e.g. 'weconnect://...')", redirectURL.Scheme)
	}

	return parseAuthLocation(redirectURL)
}

func resolveLocation(base *url.URL, location string) (*url.URL, error) {
	locURL, err := url.Parse(location)
	if err != nil {
		return nil, err
	}
	if locURL.IsAbs() {
		return locURL, nil
	}
	return base.ResolveReference(locURL), nil
}

func parseAuthLocation(u *url.URL) (url.Values, error) {
	if u.Fragment != "" {
		u.RawQuery = u.Fragment
	}

	if errStr := u.Query().Get("error"); errStr != "" {
		return nil, errors.New(errStr)
	}

	if consent := u.Query().Get("updated") != "" || strings.Contains(u.Path, "/consent/"); consent {
		return nil, api.UrlError(
			fmt.Sprintf("terms of service updated- please open app or website and confirm: %s", u.String()),
			u,
		)
	}

	return u.Query(), nil
}

func extractState(body []byte) (string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	stateInput := doc.Find("input[name=state]").First()
	if stateInput.Length() == 0 {
		return "", errors.New("state parameter not found")
	}

	state, ok := stateInput.Attr("value")
	if !ok || state == "" {
		return "", errors.New("state value not found")
	}

	return state, nil
}

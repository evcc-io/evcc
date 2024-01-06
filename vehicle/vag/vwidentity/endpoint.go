package vwidentity

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
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
		"nonce": []string{lo.RandomString(43, lo.LettersCharset)},
		"state": []string{uuid.NewString()},
	}

	var vars FormVars
	uri = uri + "&" + query.Encode()

	// GET identity.vwgroup.io/signin-service/v1/signin/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com?relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa
	resp, err := v.Get(uri)
	if err == nil {
		vars, err = FormValues(resp.Body, "form#emailPasswordForm")
		resp.Body.Close()
	}

	var params CredentialParams

	// POST identity.vwgroup.io/signin-service/v1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com/login/identifier
	if err == nil {
		data := url.Values{
			"_csrf":      {vars.Inputs["_csrf"]},
			"relayState": {vars.Inputs["relayState"]},
			"hmac":       {vars.Inputs["hmac"]},
			"email":      {user},
		}

		uri = BaseURL + vars.Action
		if resp, err = v.PostForm(uri, data); err == nil {
			if params, err = ParseCredentialsPage(resp.Body); err == nil && params.TemplateModel.Error != "" {
				err = errors.New(params.TemplateModel.Error)
			}
			resp.Body.Close()
		}
	}

	// POST identity.vwgroup.io/signin-service/v1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com/login/authenticate
	if err == nil {
		data := url.Values{
			"_csrf":      {params.CsrfToken},
			"relayState": {params.TemplateModel.RelayState},
			"hmac":       {params.TemplateModel.Hmac},
			"email":      {user},
			"password":   {password},
		}

		// reuse url from identifier step before
		uri = strings.ReplaceAll(uri, params.TemplateModel.IdentifierUrl, params.TemplateModel.PostAction)

		if resp, err = v.PostForm(uri, data); err == nil {
			resp.Body.Close()

			if resp.StatusCode >= http.StatusBadRequest {
				err = errors.New(resp.Status)
			}
		}

		if err == nil {
			if e := resp.Request.URL.Query().Get("error"); e != "" {
				err = fmt.Errorf(e)
			}

			if u := resp.Request.URL.Query().Get("updated"); err == nil && u != "" {
				err = errors.New("terms of service updated- please open app or website and confirm")
			}
		}
	}

	// GET identity.vwgroup.io/oidc/v1/oauth/sso?clientId=b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com&relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa&userId=bca09cc0-8eba-4110-af71-7242868e1bf1&HMAC=2b01ce6a351fad4dd97dc8110d0967b46c95889ab5010c660a616462e66a83ca
	// GET identity.vwgroup.io/signin-service/v1/consent/users/bca09cc0-8eba-4110-af71-7242868e1bf1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com?scopes=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa&callback=https://identity.vwgroup.io/oidc/v1/oauth/client/callback&hmac=a590931ca3cd9dc3a27f1d1c0c162bf1e5c5c32c9f5b40fcb36d4c6edc631e03
	// GET identity.vwgroup.io/oidc/v1/oauth/client/callback/success?user_id=bca09cc0-8eba-4110-af71-7242868e1bf1&client_id=b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com&scopes=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&consentedScopes=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&relayState=f89a0b750c93e278a7ace170ce374e9cb9eb0a74&hmac=2b728f463c3cfe80f3271fbb35680e5e5218ca70025a46e7fadf7c7982decc2b

	var location *url.URL
	if err == nil {
		loc := strings.ReplaceAll(resp.Header.Get("Location"), "#", "?") // convert to parseable url
		if location, err = url.Parse(loc); err == nil {
			return location.Query(), nil
		}
	}

	return nil, err
}

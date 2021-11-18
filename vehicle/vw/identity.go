package vw

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

const (
	// IdentityURI is the VW OIDC identity provider uri
	IdentityURI = "https://identity.vwgroup.io"

	// OauthTokenURI is used for refreshing tokens
	OauthTokenURI = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/oauth2/v1/token"

	// OauthRevokeURI is used for revoking tokens
	OauthRevokeURI = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/oauth2/v1/revoke"
)

var _ PlatformLogin = (*Identity)(nil)

// PlatformLogin provides the user login to identity.vwgroup.io. It does not handle the token exchange.
type PlatformLogin interface {
	UserLogin(uri, user, password string) (url.Values, error)
}

// TokenSourceProvider provides the token source by exchanging the platform token for specific service tokens
type TokenSourceProvider interface {
	TokenSource() (oauth2.TokenSource, error)
}

// Identity provides the identity.vwgroup.io login token source
type Identity struct {
	log *util.Logger
	*request.Helper
	oauth2.TokenSource
}

// NewIdentity creates VW identity
func NewIdentity(log *util.Logger) *Identity {
	v := &Identity{
		log:    log,
		Helper: request.NewHelper(log),
	}

	return v
}

// UserLogin performs the identity.vwgroup.io login
func (v *Identity) UserLogin(uri, user, password string) (url.Values, error) {
	if user == "" || password == "" {
		return nil, api.ErrMissingCredentials
	}

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

	var vars FormVars

	// add nonce and state
	query := url.Values{
		"nonce": []string{util.RandomString(43)},
		"state": []string{util.RandomString(43)},
	}
	uri += "&" + query.Encode()

	// GET identity.vwgroup.io/signin-service/v1/signin/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com?relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa
	resp, err := v.Get(uri)
	if err == nil {
		vars, err = FormValues(resp.Body, "form#emailPasswordForm")
		resp.Body.Close()
	}

	// POST identity.vwgroup.io/signin-service/v1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com/login/identifier
	if err == nil {
		data := url.Values(map[string][]string{
			"_csrf":      {vars.Inputs["_csrf"]},
			"relayState": {vars.Inputs["relayState"]},
			"hmac":       {vars.Inputs["hmac"]},
			"email":      {user},
		})

		uri = IdentityURI + vars.Action
		if resp, err = v.PostForm(uri, data); err == nil {
			vars, err = FormValues(resp.Body, "form#credentialsForm")
			resp.Body.Close()
		}
	}

	// POST identity.vwgroup.io/signin-service/v1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com/login/authenticate
	if err == nil {
		data := url.Values(map[string][]string{
			"_csrf":      {vars.Inputs["_csrf"]},
			"relayState": {vars.Inputs["relayState"]},
			"hmac":       {vars.Inputs["hmac"]},
			"email":      {user},
			"password":   {password},
		})

		uri = IdentityURI + vars.Action
		if resp, err = v.PostForm(uri, data); err == nil {
			resp.Body.Close()

			if e := resp.Request.URL.Query().Get("error"); e != "" {
				err = fmt.Errorf(e)
			}

			if u := resp.Request.URL.Query().Get("updated"); err == nil && u != "" {
				v.log.WARN.Println("accepting updated tos", u)
				if resp, err = v.postTos(resp.Request.URL.String()); err == nil {
					resp.Body.Close()
				}
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

func (v *Identity) postTos(uri string) (*http.Response, error) {
	var vars FormVars
	resp, err := v.Get(uri)
	if err == nil {
		vars, err = FormValues(resp.Body, "form#emailPasswordForm")
	}

	if err == nil {
		data := make(url.Values)
		for k, v := range vars.Inputs {
			data.Set(k, v)
		}

		uri := IdentityURI + vars.Action
		resp, err = v.PostForm(uri, data)
	}

	return resp, err
}

// Login executes the login flow by obtaining the oauth token source from the provider
func (v *Identity) Login(prov TokenSourceProvider) error {
	ts, err := prov.TokenSource()
	if err == nil {
		v.TokenSource = ts
	}
	return err
}

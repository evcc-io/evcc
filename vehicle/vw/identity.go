package vw

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/andig/evcc/util/request"
)

const (
	// IdentityURI is the VW OIDC identidy provider uri
	IdentityURI = "https://identity.vwgroup.io"

	// OauthTokenURI is used for refreshing tokens
	OauthTokenURI = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/oauth2/v1/token"
)

// Tokens is an OAuth tokens response
type Tokens struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// OIDCResponse is the well-known OIDC provider response
// https://{oauth-provider-hostname}/.well-known/openid-configuration
type OIDCResponse struct {
	Issuer      string   `json:"issuer"`
	AuthURL     string   `json:"authorization_endpoint"`
	TokenURL    string   `json:"token_endpoint"`
	JWKSURL     string   `json:"jwks_uri"`
	UserInfoURL string   `json:"userinfo_endpoint"`
	Algorithms  []string `json:"id_token_signing_alg_values_supported"`
}

// Identity provides the identity.vwgroup.io login
type Identity struct {
	*http.Client
}

func (v *Identity) redirect(resp *http.Response, err error) (*http.Response, error) {
	if err == nil {
		uri := resp.Header.Get("Location")
		resp, err = v.Get(uri)
	}

	return resp, err
}

// Login performs the identity.vwgroup.io login
func (v *Identity) Login(uri, user, password string) (*http.Response, error) {
	var vars FormVars
	var req *http.Request

	// GET identity.vwgroup.io/oidc/v1/authorize?ui_locales=de&scope=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&response_type=code&state=gmiJOaB4&redirect_uri=https%3A%2F%2Fwww.portal.volkswagen-we.com%2Fportal%2Fweb%2Fguest%2Fcomplete-login&nonce=38042ee3-b7a7-43cf-a9c1-63d2f3f2d9f3&prompt=login&client_id=b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com
	resp, err := v.Get(uri)

	// GET identity.vwgroup.io/signin-service/v1/signin/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com?relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa
	if err == nil {
		uri = resp.Header.Get("Location")
		resp, err = v.Get(uri)

		if err == nil {
			vars, err = FormValues(resp.Body, "form#emailPasswordForm")
		}
	}

	// POST identity.vwgroup.io/signin-service/v1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com/login/identifier
	if err == nil {
		data := url.Values(map[string][]string{
			"_csrf":      {vars.Csrf},
			"relayState": {vars.RelayState},
			"hmac":       {vars.Hmac},
			"email":      {user},
		})

		uri = IdentityURI + vars.Action
		req, err = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)

		if err == nil {
			resp, err = v.Do(req)
		}
	}

	// GET identity.vwgroup.io/signin-service/v1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com/login/authenticate?relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa&email=...
	if err == nil {
		uri = IdentityURI + resp.Header.Get("Location")
		req, err = http.NewRequest(http.MethodGet, uri, nil)

		if err == nil {
			resp, err = v.Do(req)
		}

		if err == nil {
			vars, err = FormValues(resp.Body, "form#credentialsForm")
		}
	}

	// POST identity.vwgroup.io/signin-service/v1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com/login/authenticate
	if err == nil {
		data := url.Values(map[string][]string{
			"_csrf":      {vars.Csrf},
			"relayState": {vars.RelayState},
			"hmac":       {vars.Hmac},
			"email":      {user},
			"password":   {password},
		})

		uri = IdentityURI + vars.Action
		req, err = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)

		if err == nil {
			resp, err = v.Do(req)
		}
	}

	// GET identity.vwgroup.io/oidc/v1/oauth/sso?clientId=b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com&relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa&userId=bca09cc0-8eba-4110-af71-7242868e1bf1&HMAC=2b01ce6a351fad4dd97dc8110d0967b46c95889ab5010c660a616462e66a83ca
	// GET identity.vwgroup.io/signin-service/v1/consent/users/bca09cc0-8eba-4110-af71-7242868e1bf1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com?scopes=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa&callback=https://identity.vwgroup.io/oidc/v1/oauth/client/callback&hmac=a590931ca3cd9dc3a27f1d1c0c162bf1e5c5c32c9f5b40fcb36d4c6edc631e03
	// GET identity.vwgroup.io/oidc/v1/oauth/client/callback/success?user_id=bca09cc0-8eba-4110-af71-7242868e1bf1&client_id=b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com&scopes=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&consentedScopes=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&relayState=f89a0b750c93e278a7ace170ce374e9cb9eb0a74&hmac=2b728f463c3cfe80f3271fbb35680e5e5218ca70025a46e7fadf7c7982decc2b
	for i := 6; i < 9; i++ {
		resp, err = v.redirect(resp, err)
	}

	return resp, err
}

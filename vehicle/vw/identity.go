package vw

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/oidc"
	"golang.org/x/net/publicsuffix"
)

const (
	// IdentityURI is the VW OIDC identidy provider uri
	IdentityURI = "https://identity.vwgroup.io"

	// OauthTokenURI is used for refreshing tokens
	OauthTokenURI = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/oauth2/v1/token"

	// OauthRevokeURI is used for revoking tokens
	OauthRevokeURI = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/oauth2/v1/revoke"
)

// Identity provides the identity.vwgroup.io login
type Identity struct {
	log *util.Logger
	*request.Helper
	clientID string
	tokens   oidc.Tokens
}

// NewIdentity creates VW identity
func NewIdentity(log *util.Logger, clientID string) *Identity {
	v := &Identity{
		log:      log,
		Helper:   request.NewHelper(log),
		clientID: clientID,
	}

	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	// track cookies and don't follow redirects
	v.Client.Jar = jar
	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return v
}

// redirect follows HTTP redirect header if error is nil. Request body is closed.
func (v *Identity) redirect(resp *http.Response, err error) (*http.Response, error) {
	if err == nil {
		uri := resp.Header.Get("Location")
		if uri == "" {
			return nil, errors.New("could not find expected HTTP redirect header\ngo to https://www.portal.volkswagen-we.com/ check account status")
		}

		if resp, err = v.Get(uri); err == nil {
			resp.Body.Close()
		}
	}

	return resp, err
}

// Login performs the identity.vwgroup.io login
func (v *Identity) Login(query url.Values, user, password string) error {
	var vars FormVars
	var req *http.Request

	// add nonce and state
	query.Set("nonce", RandomString(43))
	query.Set("state", RandomString(43))

	// GET identity.vwgroup.io/oidc/v1/authorize?ui_locales=de&scope=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&response_type=code&state=gmiJOaB4&redirect_uri=https%3A%2F%2Fwww.portal.volkswagen-we.com%2Fportal%2Fweb%2Fguest%2Fcomplete-login&nonce=38042ee3-b7a7-43cf-a9c1-63d2f3f2d9f3&prompt=login&client_id=b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com
	uri := "https://identity.vwgroup.io/oidc/v1/authorize?" + query.Encode()
	resp, err := v.Get(uri)
	if err == nil {
		resp.Body.Close()
	}

	// ID - get login url (previous request is ignored)
	if v.clientID == "" {
		uri := "https://login.apps.emea.vwapps.io/authorize?nonce=NZ2Q3T6jak0E5pDh&redirect_uri=weconnect://authenticated"
		if resp, err = v.Get(uri); err == nil {
			resp.Body.Close()

			uri = resp.Header.Get("Location")
			if resp, err = v.Get(uri); err == nil {
				resp.Body.Close()
			}
		}
	}

	// GET identity.vwgroup.io/signin-service/v1/signin/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com?relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa
	if err == nil {
		uri = resp.Header.Get("Location")
		resp, err = v.Get(uri)

		if err == nil {
			vars, err = FormValues(resp.Body, "form#emailPasswordForm")
			resp.Body.Close()
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
		if resp, err = v.PostForm(uri, data); err == nil {
			resp.Body.Close()
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
			resp.Body.Close()
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
		if resp, err = v.PostForm(uri, data); err == nil {
			resp.Body.Close()
		}
	}

	// GET identity.vwgroup.io/oidc/v1/oauth/sso?clientId=b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com&relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa&userId=bca09cc0-8eba-4110-af71-7242868e1bf1&HMAC=2b01ce6a351fad4dd97dc8110d0967b46c95889ab5010c660a616462e66a83ca
	// GET identity.vwgroup.io/signin-service/v1/consent/users/bca09cc0-8eba-4110-af71-7242868e1bf1/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com?scopes=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa&callback=https://identity.vwgroup.io/oidc/v1/oauth/client/callback&hmac=a590931ca3cd9dc3a27f1d1c0c162bf1e5c5c32c9f5b40fcb36d4c6edc631e03
	// GET identity.vwgroup.io/oidc/v1/oauth/client/callback/success?user_id=bca09cc0-8eba-4110-af71-7242868e1bf1&client_id=b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com&scopes=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&consentedScopes=openid%20profile%20birthdate%20nickname%20address%20phone%20cars%20mbb&relayState=f89a0b750c93e278a7ace170ce374e9cb9eb0a74&hmac=2b728f463c3cfe80f3271fbb35680e5e5218ca70025a46e7fadf7c7982decc2b
	for i := 6; i < 9; i++ {
		resp, err = v.redirect(resp, err)
	}

	var location *url.URL

	if err == nil {
		loc := strings.ReplaceAll(resp.Header.Get("Location"), "#", "?") //  convert to parseable url
		location, err = url.Parse(loc)

		if err == nil && location.Query().Get("id_token") == "" {
			err = errors.New("missing id token")
		}
	}

	// VW or Audi
	if err == nil && v.clientID != "" {
		data := url.Values(map[string][]string{
			"grant_type": {"id_token"},
			"scope":      {"sc2:fal"},
			"token":      {location.Query().Get("id_token")},
		})

		req, err = request.New(http.MethodPost, OauthTokenURI, strings.NewReader(data.Encode()), map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"X-Client-Id":  v.clientID,
		})

		if err == nil {
			var tokens oidc.Tokens
			if err = v.DoJSON(req, &tokens); err == nil {
				err = v.validateTokens(tokens)
			}
		}
	}

	// ID
	if err == nil && v.clientID == "" {
		q := location.Query()

		data := map[string]string{
			"state":             q.Get("state"),
			"id_token":          q.Get("id_token"),
			"redirect_uri":      "weconnect://authenticated",
			"region":            "emea",
			"access_token":      q.Get("access_token"),
			"authorizationCode": q.Get("code"),
		}

		uri := "https://login.apps.emea.vwapps.io/login/v1"
		req, err = request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

		if err == nil {
			var tokens idTokens
			if err = v.DoJSON(req, &tokens); err == nil {
				err = v.validateTokens(tokens.AsOIDC())
			}
		}
	}

	return err
}

// validateTokens checks if token is present and sets valid time
func (v *Identity) validateTokens(tokens oidc.Tokens) error {
	if tokens.AccessToken == "" {
		return errors.New("missing access token")
	}

	v.tokens.AccessToken = tokens.AccessToken
	v.tokens.Valid = time.Now().Add(time.Second * time.Duration(tokens.ExpiresIn))

	// re-use refresh token
	if tokens.RefreshToken != "" {
		v.tokens.RefreshToken = tokens.RefreshToken
	}

	return nil
}

// Token returns the access token, refreshed if necessary
func (v *Identity) Token() string {
	// give some extra time of 1m to safely trigger new tokens before they expire
	if time.Until(v.tokens.Valid) < time.Minute {
		if err := v.RefreshToken(); err != nil {
			v.log.ERROR.Printf("token refresh failed: %v", err)
		}
	}

	return v.tokens.AccessToken
}

// RefreshToken uses the refresh token to obtain a new access token
func (v *Identity) RefreshToken() error {
	if v.tokens.RefreshToken == "" {
		return errors.New("missing refresh token")
	}

	if v.clientID == "" {
		return v.refreshIDToken()
	}

	data := url.Values(map[string][]string{
		"grant_type":    {"refresh_token"},
		"refresh_token": {v.tokens.RefreshToken},
		"scope":         {"sc2:fal"},
	})

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"X-Client-Id":  v.clientID,
	}

	req, err := request.New(http.MethodPost, OauthTokenURI, strings.NewReader(data.Encode()), headers)

	if err == nil {
		var tokens oidc.Tokens
		if err = v.DoJSON(req, &tokens); err == nil {
			err = v.validateTokens(tokens)
		}
	}

	return err
}

func (v *Identity) refreshIDToken() error {
	uri := "https://login.apps.emea.vwapps.io/refresh/v1"

	headers := map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + v.tokens.RefreshToken,
	}

	req, err := request.New(http.MethodGet, uri, nil, headers)

	if err == nil {
		var tokens idTokens
		if err = v.DoJSON(req, &tokens); err == nil {
			err = v.validateTokens(tokens.AsOIDC())
		}
	}

	return err
}

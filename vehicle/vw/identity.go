package vw

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/id"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

const (
	// IdentityURI is the VW OIDC identidy provider uri
	IdentityURI = "https://identity.vwgroup.io"

	// OauthTokenURI is used for refreshing tokens
	OauthTokenURI = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/oauth2/v1/token"

	// OauthRevokeURI is used for revoking tokens
	OauthRevokeURI = "https://mbboauth-1d.prd.ece.vwg-connect.com/mbbcoauth/mobile/oauth2/v1/revoke"
)

// Identity provides the identity.vwgroup.io login token source
type Identity struct {
	log *util.Logger
	*request.Helper
	clientID string
	oauth2.TokenSource
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
		if req.URL.Scheme != "https" {
			return http.ErrUseLastResponse
		}
		return nil
	}

	return v
}

// Login performs the identity.vwgroup.io login
func (v *Identity) Login(query url.Values, user, password string) error {
	var vars FormVars
	var req *http.Request

	// add nonce and state
	query.Set("nonce", RandomString(43))
	query.Set("state", RandomString(43))

	uri := "https://identity.vwgroup.io/oidc/v1/authorize?" + query.Encode()

	// ID - get login url
	if v.clientID == "" {
		uri = "https://login.apps.emea.vwapps.io/authorize?nonce=NZ2Q3T6jak0E5pDh&redirect_uri=weconnect://authenticated"
	}

	resp, err := v.Get(uri)
	if err == nil {
		resp.Body.Close()
	}

	// GET identity.vwgroup.io/signin-service/v1/signin/b7a5bb47-f875-47cf-ab83-2ba3bf6bb738@apps_vw-dilab_com?relayState=15404cb51c8b4cc5efeee1d2c2a73e5b41562faa
	if err == nil {
		vars, err = FormValues(resp.Body, "form#emailPasswordForm")
		resp.Body.Close()
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
		vars, err = FormValues(resp.Body, "form#credentialsForm")
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
			var token Token
			if err = v.DoJSON(req, &token); err == nil {
				v.TokenSource = token.TokenSource(v.log, v.clientID)
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
			var token id.Token
			if err = v.DoJSON(req, &token); err == nil {
				v.TokenSource = token.TokenSource(v.log)
			}
		}
	}

	return err
}

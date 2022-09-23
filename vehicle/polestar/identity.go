package polestar

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.polestar

const OAuthURI = "https://polestarid.eu.polestar.com"

// https://polestarid.eu.polestar.com/.well-known/openid-configuration
var OAuth2Config = &oauth2.Config{
	ClientID:    "polxplore",
	RedirectURL: "polestar-explore://explore.polestar.com",
	Endpoint: oauth2.Endpoint{
		AuthURL:  OAuthURI + "/as/authorization.oauth2",
		TokenURL: OAuthURI + "/as/token.oauth2",
	},
	Scopes: []string{"openid", "profile", "email", "customer:attributes"}, // "oidc.profile.read",
	// "energy:battery_charge_level", "energy:charging_connection_status", "energy:charging_system_status",
	// "energy:electric_range", "energy:estimated_charging_time",
	// "exve:odometer_status", "vehicle:capabilities", "vehicle:climatization_status", "vehicle:climatization",

}

type Identity struct {
	*request.Helper
	oc *oauth2.Config
	oauth2.TokenSource
}

// NewIdentity creates Mercedes Benz identity
func NewIdentity(log *util.Logger, oc *oauth2.Config) *Identity {
	return &Identity{
		Helper: request.NewHelper(log),
		oc:     oc,
	}
}

// github.com/uhthomas/tesla
func state() string {
	var b [9]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func (v *Identity) Login(user, password string) error {
	if v.Client.Jar == nil {
		var err error
		v.Client.Jar, err = cookiejar.New(&cookiejar.Options{
			PublicSuffixList: publicsuffix.List,
		})
		if err != nil {
			return err
		}
	}

	cv, err := cv.CreateCodeVerifier()
	if err != nil {
		return err
	}

	uri := v.oc.AuthCodeURL(state(), oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", cv.CodeChallengeS256()),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("access_token_manager_id", "JWTpolxplore"),
	)

	var resume string
	if err == nil {
		var param request.InterceptResult
		v.Client.CheckRedirect, param = request.InterceptRedirect("resumePath", false)

		if _, err = v.Get(uri); err == nil {
			resume, err = param()
		}

		v.Client.CheckRedirect = nil
	}

	var code string
	if err == nil {
		params := url.Values{
			"pf.username": []string{user},
			"pf.pass":     []string{password},
		}

		var param request.InterceptResult
		v.Client.CheckRedirect, param = request.InterceptRedirect("code", true)

		uri = fmt.Sprintf("%s/as/%s/resume/as/authorization.ping?client_id=%s", OAuthURI, resume, OAuth2Config.ClientID)

		if _, err = v.Post(uri, request.FormContent, strings.NewReader(params.Encode())); err == nil {
			code, err = param()
		}

		v.Client.CheckRedirect = nil
	}

	var token oauth.Token
	if err == nil {
		params := url.Values{
			"code":          []string{code},
			"code_verifier": []string{cv.CodeChallengePlain()},
			"redirect_uri":  []string{OAuth2Config.RedirectURL},
			"grant_type":    []string{"authorization_code"},
		}

		var req *http.Request
		req, err = request.New(http.MethodPost, OAuth2Config.Endpoint.TokenURL, strings.NewReader(params.Encode()), map[string]string{
			"Content-Type":  request.FormContent,
			"Authorization": "Basic cG9seHBsb3JlOlhhaUtvb0hlaXJlaXNvb3NhaDBFdjZxdW9oczhjb2hGZUtvaHdpZTFhZTdraWV3b2hkb295ZWk5QWVZZWlXb2g",
		})
		if err == nil {
			err = v.DoJSON(req, &token)
		}
	}

	if err == nil {
		v.TokenSource = oauth.RefreshTokenSource((*oauth2.Token)(&token), v)
	}

	return err
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	return nil, errors.New("foo")

	// data := struct {
	// 	AccessToken  string `json:"accessToken"`
	// 	RefreshToken string `json:"refreshToken"`
	// }{
	// 	AccessToken:  token.AccessToken,
	// 	RefreshToken: token.RefreshToken,
	// }

	// uri := fmt.Sprintf("%s/%s", API, "accounts/refresh_token")
	// req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	// var res *oauth2.Token
	// if err == nil {
	// 	var refreshed Token
	// 	if err = c.DoJSON(req, &refreshed); err == nil {
	// 		res = refreshed.AsOAuth2Token()
	// 	}
	// }

	// return res, err
}

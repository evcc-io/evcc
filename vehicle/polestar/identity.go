package polestar

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/hasura/go-graphql-client"
	"github.com/samber/lo"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.polestar

const OAuthURI = "https://polestarid.eu.polestar.com"

// https://polestarid.eu.polestar.com/.well-known/openid-configuration
var OAuth2Config = &oauth2.Config{
	ClientID:    "l3oopkc_10",
	RedirectURL: "https://www.polestar.com/sign-in-callback",
	Endpoint: oauth2.Endpoint{
		AuthURL:  OAuthURI + "/as/authorization.oauth2",
		TokenURL: OAuthURI + "/as/token.oauth2",
	},
	Scopes: []string{
		"openid", "profile", "email", "customer:attributes",
		// "conve:recharge_status", "conve:fuel_status", "conve:odometer_status",
		// "energy:charging_connection_status", "energy:electric_range", "energy:estimated_charging_time", "energy:recharge_status",
		// "energy:battery_charge_level", "energy:charging_system_status", "energy:charging_timer", "energy:electric_range", "energy:recharge_status",
		// "energy:battery_charge_level",
	},
}

type Identity struct {
	*request.Helper
	user, password string
}

// NewIdentity creates Polestar identity
func NewIdentity(log *util.Logger, user, password string) (oauth2.TokenSource, error) {
	v := &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
	}

	v.Client.Jar, _ = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	token, err := v.login()

	return oauth.RefreshTokenSource(token, v), err
}

func (v *Identity) login() (*oauth2.Token, error) {
	state := lo.RandomString(16, lo.AlphanumericCharset)
	uri := OAuth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	var param request.InterceptResult
	v.Client.CheckRedirect, param = request.InterceptRedirect("resumePath", true)
	defer func() { v.Client.CheckRedirect = nil }()

	if _, err := v.Get(uri); err != nil {
		return nil, err
	}

	resume, err := param()
	if err != nil {
		return nil, err
	}

	params := url.Values{
		"pf.username": []string{v.user},
		"pf.pass":     []string{v.password},
	}

	uri = fmt.Sprintf("%s/as/%s/resume/as/authorization.ping?client_id=%s", OAuthURI, resume, OAuth2Config.ClientID)

	var code string
	var uid string
	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		code = req.URL.Query().Get("code")
		uid = req.URL.Query().Get("uid")
		return nil
	}
	defer func() { v.Client.CheckRedirect = nil }()

	if _, err := v.Post(uri, request.FormContent, strings.NewReader(params.Encode())); err != nil {
		return nil, err
	}

	// If the authorization code is empty, this indicates that user consent must be handled
	// before the code can be obtained. The `confirmConsentAndGetCode` method is called as a
	// workaround to guide the user through the consent process and retrieve the authorization code.
	if code == "" {
		code, err = v.confirmConsentAndGetCode(resume, uid)
		if err != nil {
			return nil, err
		}
	}

	var res struct {
		Token `graphql:"getAuthToken(code: $code)"`
	}

	if err := graphql.NewClient(ApiURI+"/auth", v.Client).
		Query(context.Background(), &res, map[string]any{
			"code": code,
		}, graphql.OperationName("getAuthToken")); err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(res.ExpiresIn) * time.Second),
	}

	return token, err
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	var res struct {
		Token `graphql:"refreshAuthToken(token: $token)"`
	}

	err := graphql.NewClient(ApiURI+"/auth", v.Client).WithRequestModifier(func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	}).Query(context.Background(), &res, map[string]any{
		"token": token.RefreshToken,
	}, graphql.OperationName("refreshAuthToken"))

	if err == nil {
		return &oauth2.Token{
			AccessToken:  res.AccessToken,
			RefreshToken: res.RefreshToken,
			Expiry:       time.Now().Add(time.Duration(res.ExpiresIn) * time.Second),
		}, nil
	}

	return v.login()
}

func (v *Identity) confirmConsentAndGetCode(resume, uid string) (string, error) {
	// Extract the user ID (UID) from the redirect parameters
	if uid == "" {
		return "", fmt.Errorf("failed to extract user ID")
	}

	// Confirm user consent by submitting the consent form, which rejects cookies
	data := url.Values{
		"pf.submit": []string{"true"},
		"subject":   []string{uid},
	}

	// Retrieve the authorization code after consent has been confirmed
	var param request.InterceptResult
	v.Client.CheckRedirect, param = request.InterceptRedirect("code", true)
	defer func() { v.Client.CheckRedirect = nil }()

	// Make a POST request to confirm the user consent
	if _, err := v.Post(fmt.Sprintf("%s/as/%s/resume/as/authorization.ping", OAuthURI, resume), request.FormContent, strings.NewReader(data.Encode())); err != nil {
		return "", fmt.Errorf("failed confirming user consent: %w", err)
	}

	// Extract the authorization code from the response
	code, err := param()
	if err != nil || code == "" {
		return "", fmt.Errorf("failed extracting authorisation code: %w", err)
	}

	// Return the retrieved code
	return code, nil
}

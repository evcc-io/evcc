package polestar

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/hasura/go-graphql-client"
	"github.com/samber/lo"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.polestar

const (
	OAuthURI  = "https://polestarid.eu.polestar.com"
	basicAuth = "cG9seHBsb3JlOlhhaUtvb0hlaXJlaXNvb3NhaDBFdjZxdW9oczhjb2hGZUtvaHdpZTFhZTdraWV3b2hkb295ZWk5QWVZZWlXb2g"
)

// https://polestarid.eu.polestar.com/.well-known/openid-configuration
var OAuth2Config = &oauth2.Config{
	// ClientID:    "polxplore",
	// RedirectURL: "polestar-explore://explore.polestar.com",
	ClientID:    "polmystar",
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
	oauth2.TokenSource
}

// NewIdentity creates Polestar identity
func NewIdentity(log *util.Logger) *Identity {
	return &Identity{
		Helper: request.NewHelper(log),
	}
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

	cv := oauth2.GenerateVerifier()

	state := lo.RandomString(16, lo.AlphanumericCharset)
	uri := OAuth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(cv))
	fmt.Println(uri)

	var param request.InterceptResult
	v.Client.CheckRedirect, param = request.InterceptRedirect("resumePath", false)
	defer func() { v.Client.CheckRedirect = nil }()

	if _, err := v.Get(uri); err != nil {
		fmt.Println(uri)
		fmt.Println(1)
		return err
	}

	resume, err := param()
	if err != nil {
		fmt.Println(2)
		return err
	}

	params := url.Values{
		"pf.username": []string{user},
		"pf.pass":     []string{password},
	}

	uri = fmt.Sprintf("%s/as/%s/resume/as/authorization.ping?client_id=%s", OAuthURI, resume, OAuth2Config.ClientID)
	v.Client.CheckRedirect, param = request.InterceptRedirect("code", true)
	defer func() { v.Client.CheckRedirect = nil }()

	var code string
	if _, err = v.Post(uri, request.FormContent, strings.NewReader(params.Encode())); err == nil {
		code, err = param()
	}

	if err != nil {
		return err
	}

	gqlClient := graphql.NewClient("https://pc-api.polestar.com/eu-north-1/auth", v.Client)

	type Token struct {
		// Access_Token string
		IdToken      string `graphql:"id_token"`
		AccessToken  string `graphql:"access_token"`
		RefreshToken string `graphql:"refresh_token"`
		ExpiresIn    int    `graphql:"expires_in"`
	}

	var res struct {
		Token `graphql:"getAuthToken(code: $code)"`
	}

	// if err := gqlClient.WithRequestModifier(func(r *http.Request) {
	// 	r.Method = http.MethodOptions
	// }).Query(context.Background(), &res, map[string]any{
	// 	"code": code,
	// }, graphql.OperationName("getAuthToken")); err != nil {
	// 	// return err
	// }

	if err := gqlClient.Query(context.Background(), &res, map[string]any{
		"code": code,
	}, graphql.OperationName("getAuthToken")); err != nil {
		return err
	}

	os.Exit(1)

	var token oauth.Token
	if err == nil {
		params := url.Values{
			"code":          []string{code},
			"code_verifier": []string{cv},
			"redirect_uri":  []string{OAuth2Config.RedirectURL},
			"grant_type":    []string{"authorization_code"},
		}

		var req *http.Request
		req, err = request.New(http.MethodPost, OAuth2Config.Endpoint.TokenURL, strings.NewReader(params.Encode()), map[string]string{
			"Content-Type":  request.FormContent,
			"Authorization": "Basic " + basicAuth,
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
	data := url.Values{
		"redirect_uri":  []string{OAuth2Config.RedirectURL},
		"refresh_token": []string{token.RefreshToken},
		"grant_type":    []string{"refresh_token"},
	}
	req, err := request.New(http.MethodPost, OAuth2Config.Endpoint.TokenURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type":  request.FormContent,
		"Authorization": "Basic " + basicAuth,
	})

	var tok oauth.Token
	if err == nil {
		if err := v.DoJSON(req, &tok); err != nil {
			return nil, err
		}
	}

	return (*oauth2.Token)(&tok), err
}

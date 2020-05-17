package vehicle

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/net/publicsuffix"
)

const (
	porscheLogin          = "https://login.porsche.com/auth/de/de_DE"
	porscheLoginAuth      = "https://login.porsche.com/auth/api/v1/de/de_DE/public/login"
	porscheAPIClientID    = "TZ4Vf5wnKeipJxvatJ60lPHYEzqZ4WNp"
	porscheAPIRedirectURI = "https://my-static02.porsche.com/static/cms/auth.html"
	porscheAPIAuth        = "https://login.porsche.com/as/authorization.oauth2"
	porscheAPIToken       = "https://login.porsche.com/as/token.oauth2"
	porscheAPI            = "https://connect-portal.porsche.com/core/api/v3/de/de_DE"
)

type porscheTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type porscheVehicleResponse struct {
	CarControlData struct {
		BatteryLevel struct {
			Unit  string
			Value float64
		}
		Mileage struct {
			Unit  string
			Value float64
		}
	}
}

// Porsche is an api.Vehicle implementation for Porsche cars
type Porsche struct {
	*embed
	*util.HTTPHelper
	user, password, vin string
	token               string
	tokenValid          time.Time
	chargeStateG        provider.FloatGetter
}

// NewPorscheFromConfig creates a new vehicle
func NewPorscheFromConfig(log *util.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{}
	util.DecodeOther(log, other, &cc)

	v := &Porsche{
		embed:      &embed{cc.Title, cc.Capacity},
		HTTPHelper: util.NewHTTPHelper(util.NewLogger("porsche")),
		user:       cc.User,
		password:   cc.Password,
		vin:        cc.VIN,
	}

	v.chargeStateG = provider.NewCached(log, v.chargeState, cc.Cache).FloatGetter()

	return v
}

// login with a my Porsche account
// looks like the backend is using a PingFederate Server with OAuth2
func (v *Porsche) login(user, password string) error {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return err
	}

	// the flow is using Oauth2 and >10 redirects
	client := &http.Client{
		Jar:     jar,
		Timeout: v.HTTPHelper.Client.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // allow >10 redirects
		},
	}

	// get the login page to get the cookies for the subsequent requests
	reqLogin, err := http.NewRequest(http.MethodGet, porscheLogin, nil)
	if err != nil {
		return err
	}

	respLogin, err := client.Do(reqLogin)
	if err != nil {
		return err
	}

	queryLogin, err := url.ParseQuery(respLogin.Request.URL.RawQuery)
	if err != nil {
		return err
	}

	sec := queryLogin.Get("sec")
	resume := queryLogin.Get("resume")
	state := queryLogin.Get("state")
	thirdPartyID := queryLogin.Get("thirdPartyId")

	dataLoginAuth := url.Values{
		"sec":          []string{sec},
		"resume":       []string{resume},
		"thirdPartyId": []string{thirdPartyID},
		"state":        []string{state},
		"username":     []string{user},
		"password":     []string{password},
		"keeploggedin": []string{"false"},
	}

	reqLoginAuth, err := http.NewRequest(http.MethodPost, porscheLoginAuth, strings.NewReader(dataLoginAuth.Encode()))
	if err != nil {
		return err
	}
	reqLoginAuth.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// process the auth so the session is authenticated
	_, err = client.Do(reqLoginAuth)
	if err != nil {
		return err
	}

	var CodeVerifier, _ = cv.CreateCodeVerifier()
	codeChallenge := CodeVerifier.CodeChallengeS256()

	dataAuth := url.Values{
		"scope":                 []string{"openid"},
		"response_type":         []string{"code"},
		"access_type":           []string{"offline"},
		"prompt":                []string{"none"},
		"client_id":             []string{porscheAPIClientID},
		"redirect_uri":          []string{porscheAPIRedirectURI},
		"code_challenge":        []string{codeChallenge},
		"code_challenge_method": []string{"S256"},
	}

	reqAPIAuth, err := http.NewRequest(http.MethodGet, porscheAPIAuth, nil)
	if err != nil {
		return err
	}
	reqAPIAuth.URL.RawQuery = dataAuth.Encode()

	respAPIAuth, err := client.Do(reqAPIAuth)
	if err != nil {
		return err
	}

	queryAPIAuth, err := url.ParseQuery(respAPIAuth.Request.URL.RawQuery)
	if err != nil {
		return err
	}

	authCode := queryAPIAuth.Get("code")

	codeVerifier := CodeVerifier.CodeChallengePlain()

	dataAPIToken := url.Values{
		"grant_type":    []string{"authorization_code"},
		"client_id":     []string{porscheAPIClientID},
		"redirect_uri":  []string{porscheAPIRedirectURI},
		"code":          []string{authCode},
		"prompt":        []string{"none"},
		"code_verifier": []string{codeVerifier},
	}

	reqAPIToken, err := http.NewRequest(http.MethodPost, porscheAPIToken, strings.NewReader(dataAPIToken.Encode()))
	if err != nil {
		return err
	}
	reqAPIToken.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	respAPIToken, err := client.Do(reqAPIToken)
	if err != nil {
		return err
	}

	b, _ := ioutil.ReadAll(respAPIToken.Body)

	var pr porscheTokenResponse
	err = json.Unmarshal(b, &pr)
	if err != nil {
		return err
	}

	if pr.AccessToken == "" || pr.ExpiresIn == 0 {
		return errors.New("could not obtain token")
	}

	v.token = pr.AccessToken
	v.tokenValid = time.Now().Add(time.Duration(pr.ExpiresIn) * time.Second)

	return nil
}

func (v *Porsche) request(uri string) (*http.Request, error) {
	if v.token == "" || time.Since(v.tokenValid) > 0 {
		if err := v.login(v.user, v.password); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err == nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", v.token))
	}

	return req, nil
}

// chargeState implements the Vehicle.ChargeState interface
func (v *Porsche) chargeState() (float64, error) {
	uri := fmt.Sprintf("%s/vehicles/%s", porscheAPI, v.vin)
	req, err := v.request(uri)
	if err != nil {
		return 0, err
	}

	var pr porscheVehicleResponse
	_, err = v.RequestJSON(req, &pr)

	return pr.CarControlData.BatteryLevel.Value, err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Porsche) ChargeState() (float64, error) {
	return v.chargeStateG()
}

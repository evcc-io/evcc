package vehicle

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
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
		RemainingRanges struct {
			ElectricalRange struct {
				Distance struct {
					Unit  string
					Value float64
				}
			}
		}
	}
}

// Porsche is an api.Vehicle implementation for Porsche cars
type Porsche struct {
	*embed
	*request.Helper
	user, password, vin string
	token               string
	tokenValid          time.Time
	chargerG            func() (interface{}, error)
}

func init() {
	registry.Add("porsche", NewPorscheFromConfig)
}

// NewPorscheFromConfig creates a new vehicle
func NewPorscheFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	log := util.NewLogger("porsche")

	var err error

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Porsche{
		embed:    &embed{cc.Title, cc.Capacity},
		Helper:   request.NewHelper(util.NewLogger("porsche")),
		user:     cc.User,
		password: cc.Password,
		vin:      strings.ToUpper(cc.VIN),
	}

	if err == nil {
		err = v.authFlow()
	}

	if err == nil && cc.VIN == "" {
		v.vin, err = findVehicle(v.vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.vin)
		}
	}

	v.chargerG = provider.NewCached(v.chargeState, cc.Cache).InterfaceGetter()

	return v, err
}

// login with a my Porsche account
// looks like the backend is using a PingFederate Server with OAuth2
func (v *Porsche) authFlow() error {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return err
	}

	// the flow is using Oauth2 and >10 redirects
	client := &http.Client{
		Jar:     jar,
		Timeout: v.Helper.Client.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // allow >10 redirects
		},
	}

	// get the login page to get the cookies for the subsequent requests
	resp, err := client.Get(porscheLogin)
	if err != nil {
		return err
	}
	resp.Body.Close()

	query, err := url.ParseQuery(resp.Request.URL.RawQuery)
	if err != nil {
		return err
	}

	sec := query.Get("sec")
	resume := query.Get("resume")
	state := query.Get("state")
	thirdPartyID := query.Get("thirdPartyId")

	dataLoginAuth := url.Values{
		"sec":          []string{sec},
		"resume":       []string{resume},
		"thirdPartyId": []string{thirdPartyID},
		"state":        []string{state},
		"username":     []string{v.user},
		"password":     []string{v.password},
		"keeploggedin": []string{"false"},
	}

	req, err := request.New(http.MethodPost, porscheLoginAuth, strings.NewReader(dataLoginAuth.Encode()), request.URLEncoding)
	if err != nil {
		return err
	}

	// process the auth so the session is authenticated
	if resp, err = client.Do(req); err != nil {
		return err
	}
	resp.Body.Close()

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

	req, err = http.NewRequest(http.MethodGet, porscheAPIAuth, nil)
	if err == nil {
		req.URL.RawQuery = dataAuth.Encode()
	}

	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	query, err = url.ParseQuery(resp.Request.URL.RawQuery)
	if err != nil {
		return err
	}

	authCode := query.Get("code")

	codeVerifier := CodeVerifier.CodeChallengePlain()

	dataAPIToken := url.Values{
		"grant_type":    []string{"authorization_code"},
		"client_id":     []string{porscheAPIClientID},
		"redirect_uri":  []string{porscheAPIRedirectURI},
		"code":          []string{authCode},
		"prompt":        []string{"none"},
		"code_verifier": []string{codeVerifier},
	}

	req, err = request.New(http.MethodPost, porscheAPIToken, strings.NewReader(dataAPIToken.Encode()), request.URLEncoding)

	var pr porscheTokenResponse
	if err == nil {
		resp, err = client.Do(req)
		if err == nil {
			err = request.DecodeJSON(resp, &pr)
		}
	}

	if pr.AccessToken == "" || pr.ExpiresIn == 0 {
		return errors.New("could not obtain token")
	}

	v.token = pr.AccessToken
	v.tokenValid = time.Now().Add(time.Duration(pr.ExpiresIn) * time.Second)

	return err
}

func (v *Porsche) request(uri string) (*http.Request, error) {
	if v.token == "" || time.Since(v.tokenValid) > 0 {
		if err := v.authFlow(); err != nil {
			return nil, err
		}
	}

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", v.token),
	})

	return req, err
}

func (v *Porsche) vehicles() (res []string, err error) {
	uri := "https://connect-portal.porsche.com/core/api/v3/de/de_DE/vehicles"
	req, err := v.request(uri)

	var vehicles []struct {
		VIN string
	}

	if err == nil {
		err = v.DoJSON(req, &vehicles)

		for _, v := range vehicles {
			res = append(res, v.VIN)
		}
	}

	return res, err
}

// chargeState implements the api.Vehicle interface
func (v *Porsche) chargeState() (interface{}, error) {
	uri := fmt.Sprintf("%s/vehicles/%s", porscheAPI, v.vin)
	req, err := v.request(uri)
	if err != nil {
		return 0, err
	}

	var pr porscheVehicleResponse
	err = v.DoJSON(req, &pr)

	return pr, err
}

// SoC implements the api.Vehicle interface
func (v *Porsche) SoC() (float64, error) {
	res, err := v.chargerG()
	if res, ok := res.(porscheVehicleResponse); err == nil && ok {
		return res.CarControlData.BatteryLevel.Value, nil
	}

	return 0, err
}

// Range implements the api.VehicleRange interface
func (v *Porsche) Range() (int64, error) {
	res, err := v.chargerG()
	if res, ok := res.(porscheVehicleResponse); err == nil && ok {
		return int64(res.CarControlData.RemainingRanges.ElectricalRange.Distance.Value), nil
	}

	return 0, err
}

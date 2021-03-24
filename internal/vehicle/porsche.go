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
	porscheAPIClientID          = "4mPO3OE5Srjb1iaUGWsbqKBvvesya8oA"
	porscheEmobilityAPIClientID = "gZLSI7ThXFB4d2ld9t8Cx2DBRvGr1zN2"
)

type porscheTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type porscheEmobilityResponse struct {
	BatteryChargeStatus struct {
		ChargeRate struct {
			Unit             string
			Value            float64
			ValueInKmPerHour int64
		}
		ChargingInDCMode                            bool
		ChargingMode                                string
		ChargingPower                               float64
		ChargingReason                              string
		ChargingState                               string
		ChargingTargetDateTime                      string
		ExternalPowerSupplyState                    string
		PlugState                                   string
		RemainingChargeTimeUntil100PercentInMinutes int64
		StateOfChargeInPercentage                   int64
		RemainingERange                             struct {
			OriginalUnit      string
			OriginalValue     int64
			Unit              string
			Value             int64
			ValueInKilometers int64
		}
	}
	ChargingStatus string
	DirectCharge   struct {
		Disabled bool
		IsActive bool
	}
	DirectClimatisation struct {
		ClimatisationState         string
		RemainingClimatisationTime int64
	}
}

// Porsche is an api.Vehicle implementation for Porsche cars
type Porsche struct {
	*embed
	*request.Helper
	user, password, vin string
	token               string
	tokenValid          time.Time
	emobiltyToken       string
	emobilityTokenValid time.Time
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

func (v *Porsche) fetchToken(client *http.Client, emobility bool) (porscheTokenResponse, error) {
	var pr porscheTokenResponse

	clientID := porscheAPIClientID
	redirectURI := "https://my.porsche.com/core/de/de_DE/"

	if emobility {
		clientID = porscheEmobilityAPIClientID
		redirectURI = "https://connect-portal.porsche.com/myservices/auth/auth.html"
	}

	var CodeVerifier, _ = cv.CreateCodeVerifier()
	codeChallenge := CodeVerifier.CodeChallengeS256()

	dataTokenAuth := url.Values{
		"redirect_uri":          []string{redirectURI},
		"client_id":             []string{clientID},
		"response_type":         []string{"code"},
		"state":                 []string{"uvobn7XJs1"},
		"scope":                 []string{"openid"},
		"access_type":           []string{"offline"},
		"country":               []string{"de"},
		"locale":                []string{"de_DE"},
		"code_challenge":        []string{codeChallenge},
		"code_challenge_method": []string{"S256"},
	}

	req, err := http.NewRequest(http.MethodGet, "https://login.porsche.com/as/authorization.oauth2", nil)
	if err != nil {
		return pr, err
	}

	req.URL.RawQuery = dataTokenAuth.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return pr, err
	}
	resp.Body.Close()

	query, err := url.ParseQuery(resp.Request.URL.RawQuery)
	if err != nil {
		return pr, err
	}

	authCode := query.Get("code")

	codeVerifier := CodeVerifier.CodeChallengePlain()

	dataAPIToken := url.Values{
		"grant_type":    []string{"authorization_code"},
		"client_id":     []string{clientID},
		"redirect_uri":  []string{redirectURI},
		"code":          []string{authCode},
		"code_verifier": []string{codeVerifier},
	}

	req, err = request.New(http.MethodPost, "https://login.porsche.com/as/token.oauth2", strings.NewReader(dataAPIToken.Encode()), request.URLEncoding)

	if err == nil {
		resp, err = client.Do(req)
		if err == nil {
			err = request.DecodeJSON(resp, &pr)
		}
	}

	if pr.AccessToken == "" || pr.ExpiresIn == 0 {
		return pr, errors.New("could not obtain token")
	}

	return pr, err
}

// login with a my Porsche account
// looks like the backend is using a PingFederate Server with OAuth2
func (v *Porsche) authFlow() error {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return err
	}

	// track cookies and follow all (>10) redirects
	client := &http.Client{
		Jar:     jar,
		Timeout: v.Helper.Client.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	// get the login page to get the cookies for the subsequent requests
	resp, err := client.Get("https://login.porsche.com/auth/de/de_DE")
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

	req, err := request.New(http.MethodPost, "https://login.porsche.com/auth/api/v1/de/de_DE/public/login", strings.NewReader(dataLoginAuth.Encode()), request.URLEncoding)
	if err != nil {
		return err
	}

	// process the auth so the session is authenticated
	if resp, err = client.Do(req); err != nil {
		return err
	}
	resp.Body.Close()

	// get the token for the generic API
	var pr porscheTokenResponse
	if pr, err = v.fetchToken(client, false); err != nil {
		return err
	}

	v.token = pr.AccessToken
	v.tokenValid = time.Now().Add(time.Duration(pr.ExpiresIn) * time.Second)

	if pr, err = v.fetchToken(client, true); err != nil {
		return err
	}

	v.emobiltyToken = pr.AccessToken
	v.emobilityTokenValid = time.Now().Add(time.Duration(pr.ExpiresIn) * time.Second)

	return err
}

func (v *Porsche) request(uri string, emobility bool) (*http.Request, error) {
	if v.token == "" || time.Since(v.tokenValid) > 0 || v.emobiltyToken == "" || time.Since(v.emobilityTokenValid) > 0 {
		if err := v.authFlow(); err != nil {
			return nil, err
		}
	}

	token := v.token
	if emobility {
		token = v.emobiltyToken
	}

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
	})

	return req, err
}

func (v *Porsche) vehicles() (res []string, err error) {
	uri := "https://connect-portal.porsche.com/core/api/v3/de/de_DE/vehicles"
	req, err := v.request(uri, false)

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
	uri := fmt.Sprintf("https://api.porsche.com/service-vehicle/de/de_DE/e-mobility/J1/%s?timezone=Europe/Berlin", v.vin)
	req, err := v.request(uri, true)
	if err != nil {
		return 0, err
	}

	req.Header.Set("apikey", porscheEmobilityAPIClientID)
	var pr porscheEmobilityResponse
	err = v.DoJSON(req, &pr)

	return pr, err
}

// SoC implements the api.Vehicle interface
func (v *Porsche) SoC() (float64, error) {
	res, err := v.chargerG()
	if res, ok := res.(porscheEmobilityResponse); err == nil && ok {
		return float64(res.BatteryChargeStatus.StateOfChargeInPercentage), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Porsche)(nil)

// Status implements the api.ChargeState interface
func (v *Porsche) Status() (api.ChargeStatus, error) {
	res, err := v.chargerG()
	if res, ok := res.(porscheEmobilityResponse); err == nil && ok {
		switch res.BatteryChargeStatus.PlugState {
		case "DISCONNECTED":
			return api.StatusA, nil
		case "CONNECTED":
			switch res.BatteryChargeStatus.ChargingState {
			case "OFF", "COMPLETED":
				return api.StatusB, nil
			case "ON":
				return api.StatusC, nil
			}
		}
	}

	return api.StatusNone, err
}

var _ api.VehicleRange = (*Porsche)(nil)

// Range implements the api.VehicleRange interface
func (v *Porsche) Range() (int64, error) {
	res, err := v.chargerG()
	if res, ok := res.(porscheEmobilityResponse); err == nil && ok {
		return int64(res.BatteryChargeStatus.RemainingERange.ValueInKilometers), nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*Porsche)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Porsche) FinishTime() (time.Time, error) {
	res, err := v.chargerG()

	if res, ok := res.(*porscheEmobilityResponse); err == nil && ok {
		t := time.Now()
		return t.Add(time.Duration(res.BatteryChargeStatus.RemainingChargeTimeUntil100PercentInMinutes) * time.Minute), err
	}

	return time.Time{}, err
}

var _ api.VehicleClimater = (*Porsche)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Porsche) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.chargerG()
	if res, ok := res.(porscheEmobilityResponse); err == nil && ok {
		switch res.DirectClimatisation.ClimatisationState {
		case "OFF":
			return false, 0, 0, nil
		case "ON":
			return true, 0, 0, nil
		}
	}

	return active, outsideTemp, targetTemp, err
}

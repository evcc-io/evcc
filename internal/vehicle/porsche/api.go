package porsche

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/net/publicsuffix"
)

type porscheTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// API is the Porsche api client
type API struct {
	log *util.Logger
	*request.Helper
	user, password          string
	clientID                string
	emobilityClientID       string
	token                   string
	tokenValid              time.Time
	emobilityTokenAvailable bool
	emobiltyToken           string
	emobilityTokenValid     time.Time
	emobilityVehicle        bool
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, clientID, emobilityClientID, user, password string) *API {
	v := &API{
		log:               log,
		Helper:            request.NewHelper(log),
		user:              user,
		password:          password,
		clientID:          clientID,
		emobilityClientID: emobilityClientID,
	}

	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	// track cookies and follow all (>10) redirects
	v.Client.Jar = jar
	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return nil
	}

	return v
}

func (v *API) fetchToken(emobility bool) (porscheTokenResponse, error) {
	var pr porscheTokenResponse

	actualClientID := v.clientID
	redirectURI := "https://my.porsche.com/core/de/de_DE/"

	if emobility {
		actualClientID = v.emobilityClientID
		redirectURI = "https://connect-portal.porsche.com/myservices/auth/auth.html"
	}

	var CodeVerifier, _ = cv.CreateCodeVerifier()
	codeChallenge := CodeVerifier.CodeChallengeS256()

	dataTokenAuth := url.Values{
		"redirect_uri":          []string{redirectURI},
		"client_id":             []string{actualClientID},
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

	resp, err := v.Client.Do(req)
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
		"client_id":     []string{actualClientID},
		"redirect_uri":  []string{redirectURI},
		"code":          []string{authCode},
		"code_verifier": []string{codeVerifier},
	}

	req, err = request.New(http.MethodPost, "https://login.porsche.com/as/token.oauth2", strings.NewReader(dataAPIToken.Encode()), request.URLEncoding)

	if err == nil {
		resp, err = v.Client.Do(req)
		if err == nil {
			err = request.DecodeJSON(resp, &pr)
		}
	}

	if pr.AccessToken == "" || pr.ExpiresIn == 0 {
		return pr, errors.New("could not obtain token")
	}

	return pr, err
}

func (v *API) Login() error {
	// get the login page to get the cookies for the subsequent requests
	resp, err := v.Client.Get("https://login.porsche.com/auth/de/de_DE")
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
	if resp, err = v.Client.Do(req); err != nil {
		return err
	}
	resp.Body.Close()

	// get the token for the generic API
	var pr porscheTokenResponse
	if pr, err = v.fetchToken(false); err != nil {
		return err
	}

	v.token = pr.AccessToken
	v.tokenValid = time.Now().Add(time.Duration(pr.ExpiresIn) * time.Second)

	if pr, err = v.fetchToken(true); err != nil {
		// we don't need to return this error, because we simply won't use the emobility API in this case
		return nil
	}

	v.emobilityTokenAvailable = true
	v.emobiltyToken = pr.AccessToken
	v.emobilityTokenValid = time.Now().Add(time.Duration(pr.ExpiresIn) * time.Second)

	return nil
}

func (v *API) request(uri string, emobilityRequest bool) (*http.Request, error) {
	if v.token == "" || time.Since(v.tokenValid) > 0 ||
		(v.emobilityVehicle && (v.emobiltyToken == "" || time.Since(v.emobilityTokenValid) > 0)) {
		if err := v.Login(); err != nil {
			return nil, err
		}
	}

	token := v.token
	if emobilityRequest {
		token = v.emobiltyToken
	}

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
	})

	return req, err
}

type Vehicle struct {
	VIN              string
	ModelDescription string
}

func (v *API) FindVehicle(vin string) (string, error) {
	uri := "https://connect-portal.porsche.com/core/api/v3/de/de_DE/vehicles"
	req, err := v.request(uri, false)

	var vehicles []Vehicle

	if err == nil {
		err = v.DoJSON(req, &vehicles)
	}

	var foundVehicle Vehicle

	if err == nil && vin == "" {
		if vin == "" && len(vehicles) == 1 {
			foundVehicle = vehicles[0]
		} else {
			for _, vehicleItem := range vehicles {
				if vehicleItem.VIN == strings.ToUpper(vin) {
					foundVehicle = vehicleItem
				}
			}
		}

		if foundVehicle.VIN == "" {
			return "", errors.New("vin not found")
		} else {
			v.log.DEBUG.Printf("found vehicle: %v", foundVehicle.VIN)
		}

		// check if the found vehicle is a Taycan, because that one supports the emobility API
		if v.emobilityTokenAvailable {
			if strings.Contains(foundVehicle.ModelDescription, "Taycan") {
				v.emobilityVehicle = true
			}
		}

	}

	return foundVehicle.VIN, err
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

// Status implements the vehicle status repsonse
func (v *API) Status(vin string) (interface{}, error) {
	if v.emobilityVehicle {
		uri := fmt.Sprintf("https://api.porsche.com/service-vehicle/de/de_DE/e-mobility/J1/%s?timezone=Europe/Berlin", vin)
		req, err := v.request(uri, true)
		if err != nil {
			return 0, err
		}

		req.Header.Set("apikey", v.emobilityClientID)
		var pr porscheEmobilityResponse
		err = v.DoJSON(req, &pr)

		return pr, err
	} else {
		uri := fmt.Sprintf("https://connect-portal.porsche.com/core/api/v3/de/de_DE/vehicles/%s", vin)
		req, err := v.request(uri, false)
		if err != nil {
			return 0, err
		}

		var pr porscheVehicleResponse
		err = v.DoJSON(req, &pr)

		return pr, err
	}
}

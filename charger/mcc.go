package charger

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
)

const (
	apiLogin                   apiFunction = "/jwt/login"
	apiRefresh                 apiFunction = "/jwt/refresh"
	apiChargeState             apiFunction = "/v1/api/WebServer/properties/chargeState"
	apiCurrentSession          apiFunction = "/v1/api/WebServer/properties/swaggerCurrentSession"
	apiEnergy                  apiFunction = "/v1/api/iCAN/properties/propjIcanEnergy"
	apiSetCurrentLimit         apiFunction = "/v1/api/SCC/properties/propHMICurrentLimit?value="
	apiCurrentCableInformation apiFunction = "/v1/api/SCC/properties/json_CurrentCableInformation"
)

// MCCErrorResponse is the API response if status not OK
type MCCErrorResponse struct {
	Error string
}

// MCCTokenResponse is the apiLogin response
type MCCTokenResponse struct {
	AccessToken string `json:"token"`
}

// MCCCurrentSession is the apiCurrentSession response
type MCCCurrentSession struct {
	Account           string     `json:"account"`
	ChargingRate      float64    `json:"chargingRate"`
	ChargingType      string     `json:"chargingType"`
	ClockSrc          string     `json:"clockSrc"`
	Costs             int64      `json:"costs"`
	Currency          string     `json:"currency"`
	DepartTime        string     `json:"departTime"`
	Duration          int64      `json:"duration"`
	EndOfChargeTime   string     `json:"endOfChargeTime"`
	EndSoc            float64    `json:"endSoc"`
	EndTime           string     `json:"endTime"`
	EnergySumKwh      float64    `json:"energySumKwh"`
	EvCharingRateKW   float64    `json:"evChargingRatekW"`
	EvTargetSoc       float64    `json:"evTargetSoc"`
	EvVasAvailability bool       `json:"evVasAvailability"`
	PcID              string     `json:"pcid"`
	PowerRange        float64    `json:"powerRange"`
	SelfEnergy        float64    `json:"selfEnergy"`
	SessionID         int64      `json:"sessionId"`
	Soc               float64    `json:"soc"`
	SolarEnergyShare  float64    `json:"solarEnergyShare"`
	StartSoc          float64    `json:"startSoc"`
	StartTime         *time.Time `json:"startTime"`
	TotalRange        float64    `json:"totalRange"`
	VehicleBrand      string     `json:"vehicleBrand"`
	VehicleModel      string     `json:"vehicleModel"`
	Whitelist         bool       `json:"whitelist"`
}

// MCCEnergyPhase is the apiEnergy response for a single phase
type MCCEnergyPhase struct {
	Ampere float64 `json:"Ampere"`
	Power  float64 `json:"Power"`
	Volts  float64 `json:"Volts"`
}

// MCCEnergy is the apiEnergy response
type MCCEnergy struct {
	L1 MCCEnergyPhase `json:"L1"`
	L2 MCCEnergyPhase `json:"L2"`
	L3 MCCEnergyPhase `json:"L3"`
}

// MCCCurrentCableInformation is the apiCurrentCableInformation response
type MCCCurrentCableInformation struct {
	CarCable     int64 `json:"carCable"`
	GridCable    int64 `json:"gridCable"`
	HwfpMaxLimit int64 `json:"hwfpMaxLimit"`
	MaxValue     int64 `json:"maxValue"`
	MinValue     int64 `json:"minValue"`
	Value        int64 `json:"value"`
}

// MCC charger implementation for supporting Mobile Charger Connect devices from Audi, Bentley, Porsche
type MCC struct {
	*api.HTTPHelper
	uri          string
	password     string
	token        string
	tokenValid   time.Time
	tokenRefresh time.Time
}

// NewMCCFromConfig creates a MCC charger from generic config
func NewMCCFromConfig(log *api.Logger, other map[string]interface{}) api.Charger {
	cc := struct{ URI, Password string }{}
	api.DecodeOther(log, other, &cc)

	return NewMCC(cc.URI, cc.Password)
}

// NewMCC creates MCC charger
func NewMCC(uri string, password string) *MCC {
	// ignore the self signed certificate
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{Transport: customTransport}

	c := &MCC{
		HTTPHelper: api.NewHTTPHelperWithClient(api.NewLogger("mcc"), client),
		uri:        uri,
		password:   password,
	}

	c.HTTPHelper.Log.WARN.Println("-- experimental --")

	return c
}

// construct the URL for a given apiFunction
func (mcc *MCC) apiURL(api apiFunction) string {
	return fmt.Sprintf("%s%s", mcc.uri, api)
}

// proces the http request to fetch the auth token for a login or refresh request
func (mcc *MCC) fetchToken(request *http.Request) error {
	var tr MCCTokenResponse
	b, err := mcc.RequestJSON(request, &tr)
	if err == nil {
		if len(tr.AccessToken) == 0 && len(b) > 0 {
			var error MCCErrorResponse

			if err := json.Unmarshal(b, &error); err != nil {
				return err
			}

			return fmt.Errorf("response: %s", error.Error)

		}

		mcc.token = tr.AccessToken
		// According to the Web Interface, the token is valid for 10 minutes
		mcc.tokenValid = time.Now().Add(time.Duration(10) * time.Minute)

		// the web interface updates the token every 2 minutes, so lets do the same here
		mcc.tokenRefresh = time.Now().Add(time.Duration(2) * time.Minute)

		return nil
	}

	return err
}

// login as the home user with the given password
func (mcc *MCC) login(password string) error {
	uri := fmt.Sprintf("%s%s", mcc.uri, apiLogin)

	data := url.Values{
		"user": []string{"user"},
		"pass": []string{mcc.password},
	}

	req, err := http.NewRequest(http.MethodPost, uri, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err = mcc.fetchToken(req)

	return err
}

// refresh the auth token with a new one
func (mcc *MCC) refresh() error {
	uri := fmt.Sprintf("%s%s", mcc.uri, apiRefresh)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf(" Bearer %s", mcc.token))

	err = mcc.fetchToken(req)

	return err
}

// creates a custom http request that contains the auth token
func (mcc *MCC) customRequest(method, uri string) (*http.Request, error) {
	// do we need to login?
	if mcc.token == "" || time.Since(mcc.tokenValid) > 0 {
		if err := mcc.login(mcc.password); err != nil {
			return nil, err
		}
	}

	// is it time to refresh the token?
	if time.Since(mcc.tokenRefresh) > 0 {
		if err := mcc.refresh(); err != nil {
			return nil, err
		}
	}

	// now lets process the request with the fetched token
	req, err := http.NewRequest(method, uri, nil)
	if err != nil {
		return req, err
	}

	req.Header.Set("Authorization", fmt.Sprintf(" Bearer %s", mcc.token))

	return req, nil
}

// use http PUT to set a value on the URI, the value should be URL encoded in the URI parameter
func (mcc *MCC) putValue(uri string) error {
	req, err := mcc.customRequest(http.MethodPut, uri)
	if err != nil {
		return err
	}

	b, err := mcc.Request(req)
	if err == nil {
		var result string
		err = json.Unmarshal(b, &result)

		if err == nil {
			if result != "OK" {
				return fmt.Errorf("Call returned an unexpected error")
			}
		}
	}

	return err
}

// use http GET to fetch a non structured value from an URI and stores it in result
func (mcc *MCC) getValue(uri string, result interface{}) error {
	req, err := mcc.customRequest(http.MethodGet, uri)
	if err != nil {
		return err
	}

	b, err := mcc.Request(req)
	if err == nil {
		err = json.Unmarshal(b, &result)
	}

	return err
}

// use http GET to fetch an escaped JSON string and unmarshall the data in result
func (mcc *MCC) getEscapedJSON(uri string, result interface{}) error {
	req, err := mcc.customRequest(http.MethodGet, uri)
	if err != nil {
		return err
	}

	b, err := mcc.Request(req)
	if err == nil {
		var unquote string
		if err := json.Unmarshal([]byte(b), &unquote); err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(unquote), &result); err != nil {
			return err
		}
	}

	return err
}

// Status implements the Charger.Status interface
func (mcc *MCC) Status() (api.ChargeStatus, error) {
	var chargeState int64

	err := mcc.getValue(mcc.apiURL(apiChargeState), &chargeState)

	if err != nil {
		return api.StatusNone, err
	}

	// Charge State values and mapping to ChargeStatus
	// 0: Unplugged		StatusA
	// 1: Connecting	StatusB
	// 2: Error				StatusE
	// 3: Established	StatusF
	// 4: Paused			StatusD
	// 5: Active			StatusC
	// 6: Finished		StatusD

	switch chargeState {
	case 0:
		return api.StatusA, nil
	case 1:
		return api.StatusB, nil
	case 2:
		return api.StatusE, nil
	case 3:
		return api.StatusF, nil
	case 4, 6:
		return api.StatusD, nil
	case 5:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("properties unknown result: %d", chargeState)
	}
}

// Enabled implements the Charger.Enabled interface
func (mcc *MCC) Enabled() (bool, error) {
	// Check if the car is connected and Paused, Active, or Finished
	var chargeState int64

	err := mcc.getValue(mcc.apiURL(apiChargeState), &chargeState)

	if err != nil {
		return false, err
	}

	if chargeState >= 4 && chargeState <= 6 {
		return true, nil
	}

	return false, nil
}

// Enable implements the Charger.Enable interface
func (mcc *MCC) Enable(enable bool) error {
	// As we don't know of the API to disable charging this for now always returns an error

	return fmt.Errorf("The device doesn't allow to enable/disable the device via an API")
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (mcc *MCC) MaxCurrent(current int64) error {
	// The device doesn't return an error if we set a value greater than the
	// current allowed max or smaller than the allowed min
	// instead it will simply set it to max or min and return "OK" anyway
	// Since the API here works differently, we fetch the limits
	// and then return an error if the value is outside of the limits or
	// otherwise set the new value
	var cableInformation MCCCurrentCableInformation
	err := mcc.getEscapedJSON(mcc.apiURL(apiCurrentCableInformation), &cableInformation)
	if err != nil {
		return err
	}

	if current < cableInformation.MinValue {
		return fmt.Errorf("value is lower than the allowed minimum value %d", cableInformation.MinValue)
	}

	if current > cableInformation.MaxValue {
		return fmt.Errorf("value is higher than the allowed maximum value %d", cableInformation.MaxValue)
	}

	url := fmt.Sprintf("%s%d", mcc.apiURL(apiSetCurrentLimit), current)
	fmt.Println(url)

	if err := mcc.putValue(url); err != nil {
		return err
	}

	return nil
}

// CurrentPower implements the Meter interface.
func (mcc *MCC) CurrentPower() (float64, error) {
	var energy MCCEnergy
	err := mcc.getEscapedJSON(mcc.apiURL(apiEnergy), &energy)

	return energy.L1.Power + energy.L2.Power + energy.L3.Power, err
}

// ChargedEnergy implements the ChargeRater interface.
func (mcc *MCC) ChargedEnergy() (float64, error) {
	var currentSession MCCCurrentSession
	err := mcc.getEscapedJSON(mcc.apiURL(apiCurrentSession), &currentSession)
	if err != nil {
		return 0, err
	}

	return currentSession.EnergySumKwh, nil
}

// ChargingTime yields current charge run duration
func (mcc *MCC) ChargingTime() (time.Duration, error) {
	var currentSession MCCCurrentSession
	err := mcc.getEscapedJSON(mcc.apiURL(apiCurrentSession), &currentSession)
	if err != nil {
		return 0, err
	}

	return time.Duration(time.Duration(currentSession.Duration) * time.Second), nil
}

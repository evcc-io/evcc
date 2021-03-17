package charger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

const (
	mccAPILogin                   = "jwt/login"
	mccAPIRefresh                 = "jwt/refresh"
	mccAPIChargeState             = "v1/api/WebServer/properties/chargeState"
	mccAPICurrentSession          = "v1/api/WebServer/properties/swaggerCurrentSession"
	mccAPIEnergy                  = "v1/api/iCAN/properties/propjIcanEnergy"
	mccAPISetCurrentLimit         = "v1/api/SCC/properties/propHMICurrentLimit?value="
	mccAPICurrentCableInformation = "v1/api/SCC/properties/json_CurrentCableInformation"
)

// MCCTokenResponse is the apiLogin response
type MCCTokenResponse struct {
	Token string
	Error string
}

// MCCCurrentSession is the apiCurrentSession response
type MCCCurrentSession struct {
	Duration     time.Duration
	EnergySumKwh float64
}

// MCCEnergyPhase is the apiEnergy response for a single phase
type MCCEnergyPhase struct {
	Ampere float64
	Power  float64
}

// MCCEnergy is the apiEnergy response
type MCCEnergy struct {
	L1, L2, L3 MCCEnergyPhase
}

// MCCCurrentCableInformation is the apiCurrentCableInformation response
type MCCCurrentCableInformation struct {
	MaxValue, MinValue, Value int64
}

// MobileConnect charger supporting devices from Audi, Bentley, Porsche
type MobileConnect struct {
	*request.Helper
	uri              string
	password         string
	token            string
	tokenExpiry      time.Time
	cableInformation MCCCurrentCableInformation
}

func init() {
	registry.Add("mcc", NewMobileConnectFromConfig)
}

// NewMobileConnectFromConfig creates a MCC charger from generic config
func NewMobileConnectFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct{ URI, Password string }{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewMobileConnect(util.DefaultScheme(cc.URI, "https"), cc.Password)
}

// NewMobileConnect creates MCC charger
func NewMobileConnect(uri string, password string) (*MobileConnect, error) {
	log := util.NewLogger("mcc")

	mcc := &MobileConnect{
		Helper:   request.NewHelper(log),
		uri:      strings.TrimRight(uri, "/"),
		password: password,
	}

	// ignore the self signed certificate
	mcc.Client.Transport = request.NewTripper(log, request.InsecureTransport())

	return mcc, nil
}

// construct the URL for a given api
func (mcc *MobileConnect) apiURL(api string) string {
	return fmt.Sprintf("%s/%s", mcc.uri, api)
}

// process the http request to fetch the auth token for a login or refresh request
func (mcc *MobileConnect) fetchToken(request *http.Request) error {
	var tr MCCTokenResponse
	err := mcc.DoJSON(request, &tr)
	if err == nil {
		if len(tr.Token) == 0 {
			return fmt.Errorf("response: %s", tr.Error)
		}

		mcc.token = tr.Token
		// According to the Web Interface, the token is valid for 2 minutes
		mcc.tokenExpiry = time.Now().Add(2 * time.Minute)
	}

	return err
}

// login as the home user with the given password
func (mcc *MobileConnect) login(password string) error {
	uri := fmt.Sprintf("%s/%s", mcc.uri, mccAPILogin)

	data := url.Values{
		"user": []string{"user"},
		"pass": []string{mcc.password},
	}

	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), map[string]string{
		"Referer":      fmt.Sprintf("%s/login", mcc.uri),
		"Content-Type": "application/x-www-form-urlencoded",
	})
	if err != nil {
		return err
	}

	return mcc.fetchToken(req)
}

// refresh the auth token with a new one
func (mcc *MobileConnect) refresh() error {
	uri := fmt.Sprintf("%s/%s", mcc.uri, mccAPIRefresh)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Referer", fmt.Sprintf("%s/login", mcc.uri))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", mcc.token))

	return mcc.fetchToken(req)
}

// creates a http request that contains the auth token
func (mcc *MobileConnect) request(method, uri string) (*http.Request, error) {
	// do we need to login?
	if mcc.token == "" {
		if err := mcc.login(mcc.password); err != nil {
			return nil, err
		}
	}

	// is it time to refresh the token?
	if time.Until(mcc.tokenExpiry) < 10*time.Second {
		if err := mcc.refresh(); err != nil {
			return nil, err
		}
	}

	// now lets process the request with the fetched token
	req, err := http.NewRequest(method, uri, nil)
	if err != nil {
		return req, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", mcc.token))
	req.Header.Set("Referer", fmt.Sprintf("%s/dashboard", mcc.uri))

	return req, nil
}

// use http GET to fetch a non structured value from an URI and stores it in result
func (mcc *MobileConnect) getValue(uri string) ([]byte, error) {
	req, err := mcc.request(http.MethodGet, uri)
	if err != nil {
		return nil, err
	}

	return mcc.DoBody(req)
}

// use http GET to fetch an escaped JSON string and unmarshal the data in result
func (mcc *MobileConnect) getEscapedJSON(uri string, result interface{}) error {
	req, err := mcc.request(http.MethodGet, uri)
	if err != nil {
		return err
	}

	b, err := mcc.DoBody(req)
	if err != nil {
		return err
	}

	s, err := strconv.Unquote(strings.Trim(string(b), "\n"))
	if err != nil {
		return err
	}

	if s == "" {
		return nil // empty response
	}

	return json.Unmarshal([]byte(s), &result)
}

// Status implements the Charger.Status interface
func (mcc *MobileConnect) Status() (api.ChargeStatus, error) {
	b, err := mcc.getValue(mcc.apiURL(mccAPIChargeState))
	if err != nil {
		return api.StatusNone, err
	}

	chargeState, err := strconv.ParseInt(strings.Trim(string(b), "\n"), 10, 8)
	if err != nil {
		return api.StatusNone, err
	}

	switch chargeState {
	case 0: // Unplugged
		return api.StatusA, nil
	case 1, 3, 4, 6: // 1: Connecting, 3: Established, 4: Paused, 6: Finished
		return api.StatusB, nil
	case 2: // Error
		return api.StatusF, nil
	case 5: // Active
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("properties unknown result: %d", chargeState)
	}
}

// Enabled implements the Charger.Enabled interface
func (mcc *MobileConnect) Enabled() (bool, error) {
	// Check if the car is connected and Paused, Active, or Finished
	b, err := mcc.getValue(mcc.apiURL(mccAPIChargeState))
	if err != nil {
		return false, err
	}

	// return value is returned in the format 0\n
	chargeState, err := strconv.ParseInt(strings.Trim(string(b), "\n"), 10, 8)
	if err != nil {
		return false, err
	}

	if chargeState >= 4 && chargeState <= 6 {
		return true, nil
	}

	return false, nil
}

// Enable implements the Charger.Enable interface
func (mcc *MobileConnect) Enable(enable bool) error {
	// As we don't know of the API to disable charging this for now always returns an error
	return nil
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (mcc *MobileConnect) MaxCurrent(current int64) error {
	// The device doesn't return an error if we set a value greater than the
	// current allowed max or smaller than the allowed min
	// instead it will simply set it to max or min and return "OK" anyway
	// Since the API here works differently, we fetch the limits
	// and then return an error if the value is outside of the limits or
	// otherwise set the new value
	if mcc.cableInformation.MaxValue == 0 {
		if err := mcc.getEscapedJSON(mcc.apiURL(mccAPICurrentCableInformation), &mcc.cableInformation); err != nil {
			return err
		}
	}

	if current < mcc.cableInformation.MinValue {
		return fmt.Errorf("value is lower than the allowed minimum value %d", mcc.cableInformation.MinValue)
	}

	if current > mcc.cableInformation.MaxValue {
		return fmt.Errorf("value is higher than the allowed maximum value %d", mcc.cableInformation.MaxValue)
	}

	url := fmt.Sprintf("%s%d", mcc.apiURL(mccAPISetCurrentLimit), current)

	req, err := mcc.request(http.MethodPut, url)
	if err != nil {
		return err
	}

	b, err := mcc.DoBody(req)
	if err != nil {
		return err
	}

	// return value is returned in the format "OK"\n
	if strings.Trim(string(b), "\n\"") != "OK" {
		return fmt.Errorf("maxcurrent unexpected response: %s", string(b))
	}

	return nil
}

// CurrentPower implements the Meter interface.
func (mcc *MobileConnect) CurrentPower() (float64, error) {
	var energy MCCEnergy
	err := mcc.getEscapedJSON(mcc.apiURL(mccAPIEnergy), &energy)

	return energy.L1.Power + energy.L2.Power + energy.L3.Power, err
}

// ChargedEnergy implements the ChargeRater interface.
func (mcc *MobileConnect) ChargedEnergy() (float64, error) {
	var currentSession MCCCurrentSession
	if err := mcc.getEscapedJSON(mcc.apiURL(mccAPICurrentSession), &currentSession); err != nil {
		return 0, err
	}

	return currentSession.EnergySumKwh, nil
}

// ChargingTime yields current charge run duration
func (mcc *MobileConnect) ChargingTime() (time.Duration, error) {
	var currentSession MCCCurrentSession
	if err := mcc.getEscapedJSON(mcc.apiURL(mccAPICurrentSession), &currentSession); err != nil {
		return 0, err
	}

	return time.Duration(currentSession.Duration * time.Second), nil
}

// Currents implements the MeterCurrent interface
func (mcc *MobileConnect) Currents() (float64, float64, float64, error) {
	var energy MCCEnergy
	err := mcc.getEscapedJSON(mcc.apiURL(mccAPIEnergy), &energy)

	return energy.L1.Ampere, energy.L2.Ampere, energy.L3.Ampere, err
}

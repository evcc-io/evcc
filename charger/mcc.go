package charger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
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
	Duration     int64
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

	wb := &MobileConnect{
		Helper:   request.NewHelper(log),
		uri:      strings.TrimRight(uri, "/"),
		password: password,
	}

	// ignore the self signed certificate
	wb.Client.Transport = request.NewTripper(log, transport.Insecure())

	return wb, nil
}

// process the http request to fetch the auth token for a login or refresh request
func (wb *MobileConnect) fetchToken(request *http.Request) error {
	var tr MCCTokenResponse
	err := wb.DoJSON(request, &tr)
	if err == nil {
		if len(tr.Token) == 0 {
			return fmt.Errorf("response: %s", tr.Error)
		}

		wb.token = tr.Token
		// According to tests, the token is valid for 10 minutes
		// but the web interface updates the token every 2 minutes, so let's enforce this
		wb.tokenExpiry = time.Now().Add(2 * time.Minute)
	}

	return err
}

// login as the home user with the given password
func (wb *MobileConnect) login(password string) error {
	data := url.Values{
		"user": []string{"user"},
		"pass": []string{wb.password},
	}

	uri := fmt.Sprintf("%s/%s", wb.uri, mccAPILogin)
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), map[string]string{
		"Referer":      fmt.Sprintf("%s/login", wb.uri),
		"Content-Type": "application/x-www-form-urlencoded",
	})
	if err != nil {
		return err
	}

	return wb.fetchToken(req)
}

// refresh the auth token with a new one
func (wb *MobileConnect) refresh() error {
	uri := fmt.Sprintf("%s/%s", wb.uri, mccAPIRefresh)
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Referer", fmt.Sprintf("%s/login", wb.uri))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", wb.token))

	return wb.fetchToken(req)
}

// creates a http request that contains the auth token
func (wb *MobileConnect) request(method, uri string) (*http.Request, error) {
	// do we need a token refresh?
	if wb.token != "" {
		// is it time to refresh the token?
		if time.Until(wb.tokenExpiry) < 10*time.Second {
			if err := wb.refresh(); err != nil {
				// if refreshing the token fails it most likely is expired
				// hence a new login is required, so let's enforce this
				// and ignore this error
				wb.token = ""
			}
		}
	}

	// do we need to login?
	if wb.token == "" {
		if err := wb.login(wb.password); err != nil {
			return nil, err
		}
	}

	// now lets process the request with the fetched token
	req, err := http.NewRequest(method, uri, nil)
	if err == nil {
		req.Header.Set("Referer", fmt.Sprintf("%s/dashboard", wb.uri))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", wb.token))
	}

	return req, err
}

// use http GET to fetch a non structured value from an URI and stores it in result
func (wb *MobileConnect) getValue(uri string) ([]byte, error) {
	req, err := wb.request(http.MethodGet, uri)
	if err != nil {
		return nil, err
	}

	return wb.DoBody(req)
}

// use http GET to fetch an escaped JSON string and unmarshal the data in result
func (wb *MobileConnect) getEscapedJSON(uri string, result interface{}) error {
	b, err := wb.getValue(uri)
	if err != nil {
		return err
	}

	s, err := strconv.Unquote(strings.Trim(string(b), "\n"))
	if err != nil || s == "" {
		// error or empty response
		return err
	}

	return json.Unmarshal([]byte(s), &result)
}

// Status implements the api.Charger interface
func (wb *MobileConnect) Status() (api.ChargeStatus, error) {
	b, err := wb.getValue(fmt.Sprintf("%s/%s", wb.uri, mccAPIChargeState))
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

// Enabled implements the api.Charger interface
func (wb *MobileConnect) Enabled() (bool, error) {
	// Check if the car is connected and Paused, Active, or Finished
	b, err := wb.getValue(fmt.Sprintf("%s/%s", wb.uri, mccAPIChargeState))
	if err != nil {
		return false, err
	}

	// return value is returned in the format 0\n
	chargeState, err := strconv.ParseInt(strings.Trim(string(b), "\n"), 10, 8)
	if err == nil && chargeState >= 4 && chargeState <= 6 {
		return true, nil
	}

	return false, err
}

// Enable implements the api.Charger interface
func (wb *MobileConnect) Enable(enable bool) error {
	// As we don't know of the API to disable charging this for now always returns an error
	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *MobileConnect) MaxCurrent(current int64) error {
	// The device doesn't return an error if we set a value greater than the
	// current allowed max or smaller than the allowed min
	// instead it will simply set it to max or min and return "OK" anyway
	// Since the API here works differently, we fetch the limits
	// and then return an error if the value is outside of the limits or
	// otherwise set the new value
	if wb.cableInformation.MaxValue == 0 {
		if err := wb.getEscapedJSON(fmt.Sprintf("%s/%s", wb.uri, mccAPICurrentCableInformation), &wb.cableInformation); err != nil {
			return err
		}
	}

	if current < wb.cableInformation.MinValue {
		return fmt.Errorf("value is lower than the allowed minimum value %d", wb.cableInformation.MinValue)
	}

	if current > wb.cableInformation.MaxValue {
		return fmt.Errorf("value is higher than the allowed maximum value %d", wb.cableInformation.MaxValue)
	}

	uri := fmt.Sprintf("%s%d", fmt.Sprintf("%s/%s", wb.uri, mccAPISetCurrentLimit), current)
	req, err := wb.request(http.MethodPut, uri)
	if err != nil {
		return err
	}

	b, err := wb.DoBody(req)
	if err == nil && strings.Trim(string(b), "\n\"") != "OK" {
		err = fmt.Errorf("maxcurrent unexpected response: %s", string(b))
	}

	return err
}

var _ api.Meter = (*MobileConnect)(nil)

// CurrentPower implements the api.Meter interface
func (wb *MobileConnect) CurrentPower() (float64, error) {
	var energy MCCEnergy
	err := wb.getEscapedJSON(fmt.Sprintf("%s/%s", wb.uri, mccAPIEnergy), &energy)
	return energy.L1.Power + energy.L2.Power + energy.L3.Power, err
}

var _ api.ChargeRater = (*MobileConnect)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *MobileConnect) ChargedEnergy() (float64, error) {
	var res MCCCurrentSession
	err := wb.getEscapedJSON(fmt.Sprintf("%s/%s", wb.uri, mccAPICurrentSession), &res)
	return res.EnergySumKwh, err
}

var _ api.ChargeTimer = (*MobileConnect)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *MobileConnect) ChargingTime() (time.Duration, error) {
	var res MCCCurrentSession
	err := wb.getEscapedJSON(fmt.Sprintf("%s/%s", wb.uri, mccAPICurrentSession), &res)
	return time.Duration(res.Duration) * time.Second, err
}

var _ api.MeterCurrent = (*MobileConnect)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *MobileConnect) Currents() (float64, float64, float64, error) {
	var res MCCEnergy
	err := wb.getEscapedJSON(fmt.Sprintf("%s/%s", wb.uri, mccAPIEnergy), &res)
	return res.L1.Ampere, res.L2.Ampere, res.L3.Ampere, err
}

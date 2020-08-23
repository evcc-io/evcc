package vehicle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/google/uuid"
	"golang.org/x/net/publicsuffix"
)

const (
	kiaURLDeviceID    = "https://prd.eu-ccapi.kia.com:8080/api/v1/spa/notifications/register"
	kiaURLCookies     = "https://prd.eu-ccapi.kia.com:8080/api/v1/user/oauth2/authorize?response_type=code&state=test&client_id=fdc85c00-0a2f-4c64-bcb4-2cfb1500730a&redirect_uri=https://prd.eu-ccapi.kia.com:8080/api/v1/user/oauth2/redirect"
	kiaURLLang        = "https://prd.eu-ccapi.kia.com:8080/api/v1/user/language"
	kiaURLLogin       = "https://prd.eu-ccapi.kia.com:8080/api/v1/user/signin"
	kiaURLAccessToken = "https://prd.eu-ccapi.kia.com:8080/api/v1/user/oauth2/token"
	kiaURLVehicles    = "https://prd.eu-ccapi.kia.com:8080/api/v1/spa/vehicles"
	kiaURLPreWakeup   = "https://prd.eu-ccapi.kia.com:8080/api/v1/spa/vehicles/"
	kiaURLSendPIN     = "https://prd.eu-ccapi.kia.com:8080/api/v1/user/pin"
	kiaURLGetStatus   = "https://prd.eu-ccapi.kia.com:8080/api/v2/spa/vehicles/"
)

// Kia is an api.Vehicle implementation with configurable getters and setters.
type Kia struct {
	*embed
	*util.HTTPHelper
	user     string
	password string
	pin      string
	chargeG  func() (float64, error)
	auth     kiaAuth
}

type kiaAuth struct {
	deviceID     string
	vehicleID    string
	controlToken string
}

type kiaBatteryResponse struct {
	ResMsg struct {
		EvStatus struct {
			BatteryStatus float64 `json:"batteryStatus"`
		} `json:"evStatus"`
	} `json:"resMsg"`
}

type vehicleIDResponse struct {
	ResMsg struct {
		Vehicles []struct {
			VehicleID string `json:"vehicleId"`
		} `json:"vehicles"`
	} `json:"resMsg"`
}

type deviceIDResponse struct {
	ResMsg struct {
		DeviceID string `json:"deviceId"`
	} `json:"resMsg"`
}

func init() {
	registry.Add("kia", NewKiaFromConfig)
}

// NewKiaFromConfig creates a new Vehicle
func NewKiaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		PIN                 string
		Cache               time.Duration
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Kia{
		embed:      &embed{cc.Title, cc.Capacity},
		HTTPHelper: util.NewHTTPHelper(util.NewLogger("kia")),
		user:       cc.User,
		password:   cc.Password,
		pin:        cc.PIN,
	}

	// api is unbelievably slowwhen retrieving status
	v.HTTPHelper.Client.Timeout = 60 * time.Second

	v.chargeG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	return v, nil
}

// request builds an HTTP request with headers and body
func (v *Kia) request(method, uri string, headers map[string]string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return req, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	return req, nil
}

// jsonRequest builds an HTTP json request with headers and body
func (v *Kia) jsonRequest(method, uri string, headers map[string]string, data interface{}) (*http.Request, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return v.request(method, uri, headers, bytes.NewReader(body))
}

// Credits to https://openwb.de/forum/viewtopic.php?f=5&t=1215&start=10#p11877

func (v *Kia) getDeviceID() (string, error) {
	uniID, _ := uuid.NewUUID()
	data := map[string]interface{}{
		"pushRegId": "1",
		"pushType":  "GCM",
		"uuid":      uniID.String(),
	}

	headers := map[string]string{
		"ccsp-service-id": "fdc85c00-0a2f-4c64-bcb4-2cfb1500730a",
		"Content-type":    "application/json;charset=UTF-8",
		"User-Agent":      "okhttp/3.10.0",
	}

	var did deviceIDResponse
	req, err := v.jsonRequest(http.MethodPost, kiaURLDeviceID, headers, data)
	if err == nil {
		_, err = v.RequestJSON(req, &did)
	}

	return did.ResMsg.DeviceID, err
}

func (v *Kia) getCookies() (cookieClient *util.HTTPHelper, err error) {
	cookieClient = util.NewHTTPHelper(v.Log)
	cookieClient.Client.Jar, err = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	if err == nil {
		_, err = cookieClient.Get(kiaURLCookies)
	}

	return cookieClient, err
}

func (v *Kia) setLanguage(cookieClient *util.HTTPHelper) error {
	headers := map[string]string{
		"Content-type": "application/json",
	}

	data := map[string]interface{}{
		"lang": "en",
	}

	req, err := v.jsonRequest(http.MethodPost, kiaURLLang, headers, data)
	if err == nil {
		_, err = cookieClient.Request(req)
	}

	return err
}

func (v *Kia) login(cookieClient *util.HTTPHelper) (string, error) {
	headers := map[string]string{
		"Content-type": "application/json",
	}

	data := map[string]interface{}{
		"email":    v.user,
		"password": v.password,
	}

	req, err := v.jsonRequest(http.MethodPost, kiaURLLogin, headers, data)
	if err != nil {
		return "", err
	}

	var redirect struct {
		RedirectURL string `json:"redirectUrl"`
	}

	var accCode string
	if _, err = cookieClient.RequestJSON(req, &redirect); err == nil {
		if parsed, err := url.Parse(redirect.RedirectURL); err == nil {
			accCode = parsed.Query().Get("code")
		}
	}

	return accCode, err
}

func (v *Kia) getToken(accCode string) (string, error) {
	headers := map[string]string{
		"Authorization": "Basic ZmRjODVjMDAtMGEyZi00YzY0LWJjYjQtMmNmYjE1MDA3MzBhOnNlY3JldA==",
		"Content-type":  "application/x-www-form-urlencoded",
		"User-Agent":    "okhttp/3.10.0",
	}

	data := url.Values{
		"grant_type":   []string{"authorization_code"},
		"redirect_uri": []string{"https%3A%2F%2Fprd.eu-ccapi.kia.com%3A8080%2Fapi%2Fv1%2Fuser%2Foauth2%2Fredirect"},
		"code":         []string{accCode},
	}

	req, err := v.request(http.MethodPost, kiaURLAccessToken, headers, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	var tokens struct {
		TokenType   string `json:"token_type"`
		AccessToken string `json:"access_token"`
	}

	var accToken string
	if _, err = v.RequestJSON(req, &tokens); err == nil {
		accToken = fmt.Sprintf("%s %s", tokens.TokenType, tokens.AccessToken)
	}

	return accToken, err
}

func (v *Kia) getVehicles(accToken, did string) (string, error) {
	headers := map[string]string{
		"Authorization":       accToken,
		"ccsp-device-id":      did,
		"ccsp-application-id": "693a33fa-c117-43f2-ae3b-61a02d24f417",
		"offset":              "1",
		"User-Agent":          "okhttp/3.10.0",
	}

	req, err := v.request(http.MethodGet, kiaURLVehicles, headers, nil)
	if err == nil {
		var vr vehicleIDResponse
		if _, err = v.RequestJSON(req, &vr); err == nil {
			if len(vr.ResMsg.Vehicles) == 1 {
				return vr.ResMsg.Vehicles[0].VehicleID, nil
			}

			err = errors.New("couldn't find vehicle")
		}
	}

	return "", err
}

func (v *Kia) preWakeup(accToken, did, vid string) error {
	data := map[string]interface{}{
		"action":   "prewakeup",
		"deviceId": did,
	}

	headers := map[string]string{
		"Authorization":       accToken,
		"ccsp-device-id":      did,
		"ccsp-application-id": "693a33fa-c117-43f2-ae3b-61a02d24f417",
		"offset":              "1",
		"Content-Type":        "application/json;charset=UTF-8",
		"User-Agent":          "okhttp/3.10.0",
	}

	req, err := v.jsonRequest(http.MethodPost, kiaURLPreWakeup+vid+"/control/engine", headers, data)
	if err == nil {
		_, err = v.Request(req)
	}

	return err
}

func (v *Kia) sendPIN(deviceID, accToken string) (string, error) {
	data := map[string]interface{}{
		"deviceId": deviceID,
		"pin":      string(v.pin),
	}

	headers := map[string]string{
		"Authorization": accToken,
		"Content-type":  "application/json;charset=UTF-8",
		"User-Agent":    "okhttp/3.10.0",
	}

	var token struct {
		ControlToken string `json:"controlToken"`
	}

	req, err := v.jsonRequest(http.MethodPut, kiaURLSendPIN, headers, data)
	if err == nil {
		_, err = v.RequestJSON(req, &token)
	}

	controlToken := ""
	if err == nil {
		controlToken = "Bearer " + token.ControlToken

	}

	return controlToken, err
}

func (v *Kia) getStatus() (float64, error) {
	headers := map[string]string{
		"Authorization":  v.auth.controlToken,
		"ccsp-device-id": v.auth.deviceID,
		"Content-Type":   "application/json",
	}

	var kr kiaBatteryResponse
	req, err := v.request(http.MethodGet, kiaURLGetStatus+v.auth.vehicleID+"/status", headers, nil)
	if err == nil {
		_, err = v.RequestJSON(req, &kr)
	}

	return kr.ResMsg.EvStatus.BatteryStatus, err
}

func (v *Kia) authFlow() (err error) {
	v.auth.deviceID, err = v.getDeviceID()

	var cookieClient *util.HTTPHelper
	if err == nil {
		cookieClient, err = v.getCookies()
	}

	if err == nil {
		err = v.setLanguage(cookieClient)
	}

	var kiaAccCode string
	if err == nil {
		kiaAccCode, err = v.login(cookieClient)
	}

	var kiaAccToken string
	if err == nil {
		kiaAccToken, err = v.getToken(kiaAccCode)
	}

	if err == nil {
		v.auth.vehicleID, err = v.getVehicles(kiaAccToken, v.auth.deviceID)
	}

	if err == nil {
		err = v.preWakeup(kiaAccToken, v.auth.deviceID, v.auth.vehicleID)
	}

	if err == nil {
		v.auth.controlToken, err = v.sendPIN(v.auth.deviceID, kiaAccToken)
	}

	return err
}

// chargeState implements the Vehicle.ChargeState interface
func (v *Kia) chargeState() (float64, error) {
	soc, err := v.getStatus()

	if err != nil && v.HTTPHelper.LastResponse().StatusCode == http.StatusUnauthorized {
		if err = v.authFlow(); err == nil {
			soc, err = v.getStatus()
		}
	}

	return soc, err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Kia) ChargeState() (float64, error) {
	return v.chargeG()
}

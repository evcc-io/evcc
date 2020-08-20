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
	kiaUrlDeviceID    = "https://prd.eu-ccapi.kia.com:8080/api/v1/spa/notifications/register"
	kiaUrlCookies     = "https://prd.eu-ccapi.kia.com:8080/api/v1/user/oauth2/authorize?response_type=code&state=test&client_id=fdc85c00-0a2f-4c64-bcb4-2cfb1500730a&redirect_uri=https://prd.eu-ccapi.kia.com:8080/api/v1/user/oauth2/redirect"
	kiaUrlLang        = "https://prd.eu-ccapi.kia.com:8080/api/v1/user/language"
	kiaUrlLogin       = "https://prd.eu-ccapi.kia.com:8080/api/v1/user/signin"
	kiaUrlAccessToken = "https://prd.eu-ccapi.kia.com:8080/api/v1/user/oauth2/token"
	kiaUrlVehicles    = "https://prd.eu-ccapi.kia.com:8080/api/v1/spa/vehicles"
	kiaUrlPreWakeup   = "https://prd.eu-ccapi.kia.com:8080/api/v1/spa/vehicles/"
	kiaUrlSendPIN     = "https://prd.eu-ccapi.kia.com:8080/api/v1/user/pin"
	kiaUrlGetStatus   = "https://prd.eu-ccapi.kia.com:8080/api/v2/spa/vehicles/"
)

// Kia is an api.Vehicle implementation with configurable getters and setters.
type Kia struct {
	*embed
	*util.HTTPHelper
	user     string
	password string
	pin      string
	chargeG  func() (float64, error)
	auth     KiaAuth
}

type KiaData struct {
	cookieClient *util.HTTPHelper
	accCode      string
	accToken     string
}

type KiaAuth struct {
	deviceId     string
	vehicleID    string
	controlToken string
	validUntil   time.Time
}

type kiaBatteryResponse struct {
	ResMsg struct {
		EvStatus struct {
			BatteryStatus float64 `json:"batteryStatus"`
		} `json:"evStatus"`
	} `json:"resMsg"`
}

type vehicleIdResponse struct {
	ResMsg struct {
		Vehicles []struct {
			VehicleId string `json:"vehicleId"`
		} `json:"vehicles"`
	} `json:"resMsg"`
}

type deviceIdResponse struct {
	ResMsg struct {
		DeviceId string `json:"deviceId"`
	} `json:"resMsg"`
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
		// "Content-Length":  "80",
		// "Host":            "prd.eu-ccapi.kia.com:8080",
		// "Connection":      "close",
		// "Accept-Encoding": "gzip, deflate",
		"User-Agent": "okhttp/3.10.0",
	}

	var did deviceIdResponse
	req, err := v.jsonRequest(http.MethodPost, kiaUrlDeviceID, headers, data)
	if err == nil {
		_, err = v.RequestJSON(req, &did)
	}

	return did.ResMsg.DeviceId, err
}

func (v *Kia) getCookies(kd *KiaData) (err error) {
	kd.cookieClient = util.NewHTTPHelper(v.Log) // re-use logger
	kd.cookieClient.Client.Jar, err = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	_, err = kd.cookieClient.Get(kiaUrlCookies)
	return err
}

func (v *Kia) setLanguage(kd *KiaData) error {
	headers := map[string]string{
		"Content-type": "application/json",
	}

	data := map[string]interface{}{
		"lang": "en",
	}

	req, err := v.jsonRequest(http.MethodPost, kiaUrlLang, headers, data)
	if err == nil {
		_, err = kd.cookieClient.Request(req)
	}

	return err
}

func (v *Kia) login(kd *KiaData) error {
	headers := map[string]string{
		"Content-type": "application/json",
	}

	data := map[string]interface{}{
		"email":    v.user,
		"password": v.password,
	}

	req, err := v.jsonRequest(http.MethodPost, kiaUrlLogin, headers, data)
	if err != nil {
		return err
	}

	var redirect struct {
		RedirectURL string `json:"redirectUrl"`
	}

	if _, err = kd.cookieClient.RequestJSON(req, &redirect); err == nil {
		if parsed, err := url.Parse(redirect.RedirectURL); err == nil {
			kd.accCode = parsed.Query().Get("code")
		}
	}

	return err
}

func (v *Kia) getToken(kd *KiaData) error {
	headers := map[string]string{
		"Authorization": "Basic ZmRjODVjMDAtMGEyZi00YzY0LWJjYjQtMmNmYjE1MDA3MzBhOnNlY3JldA==",
		"Content-type":  "application/x-www-form-urlencoded",
		// "Content-Length":  "150",
		// "Host":            "prd.eu-ccapi.kia.com:8080",
		// "Connection":      "close",
		// "Accept-Encoding": "gzip, deflate",
		"User-Agent": "okhttp/3.10.0",
	}

	data := "grant_type=authorization_code&redirect_uri=https%3A%2F%2Fprd.eu-ccapi.kia.com%3A8080%2Fapi%2Fv1%2Fuser%2Foauth2%2Fredirect&code="
	data += kd.accCode

	req, err := v.request(http.MethodPost, kiaUrlAccessToken, headers, strings.NewReader(data))
	if err != nil {
		return err
	}

	var tokens struct {
		TokenType   string `json:"token_type"`
		AccessToken string `json:"access_token"`
	}

	if _, err = v.RequestJSON(req, &tokens); err == nil {
		kd.accToken = fmt.Sprintf("%s %s", tokens.TokenType, tokens.AccessToken)
	}

	return err
}

func (v *Kia) getVehicles(kd *KiaData, did string) (string, error) {
	headers := map[string]string{
		"Authorization":       kd.accToken,
		"ccsp-device-id":      did,
		"ccsp-application-id": "693a33fa-c117-43f2-ae3b-61a02d24f417",
		"offset":              "1",
		"Host":                "prd.eu-ccapi.kia.com:8080",
		"Connection":          "close",
		"Accept-Encoding":     "gzip, deflate",
		"User-Agent":          "okhttp/3.10.0",
	}

	req, err := v.request(http.MethodGet, kiaUrlVehicles, headers, nil)
	if err == nil {
		var vr vehicleIdResponse
		if _, err = v.RequestJSON(req, &vr); err == nil {
			if len(vr.ResMsg.Vehicles) == 1 {
				return vr.ResMsg.Vehicles[0].VehicleId, nil
			}

			err = errors.New("couldn't find vehicle")
		}
	}

	return "", err
}

func (v *Kia) prewakeup(kd *KiaData, did, vid string) error {
	data := map[string]interface{}{
		"action":   "prewakeup",
		"deviceId": did,
	}

	headers := map[string]string{
		"Authorization":       kd.accToken,
		"ccsp-device-id":      did,
		"ccsp-application-id": "693a33fa-c117-43f2-ae3b-61a02d24f417",
		"offset":              "1",
		"Content-Type":        "application/json;charset=UTF-8",
		// "Content-Length":      "72",
		// "Host":                "prd.eu-ccapi.kia.com:8080",
		// "Connection":          "close",
		// "Accept-Encoding":     "gzip, deflate",
		"User-Agent": "okhttp/3.10.0",
	}

	req, err := v.jsonRequest(http.MethodPost, kiaUrlPreWakeup+vid+"/control/engine", headers, data)
	if err == nil {
		_, err = v.Request(req)
	}

	return err
}

func (v *Kia) sendPIN(auth *KiaAuth, kd KiaData) error {
	data := map[string]interface{}{
		"deviceId": auth.deviceId,
		"pin":      string(v.pin),
	}

	headers := map[string]string{
		"Authorization": kd.accToken,
		"Content-type":  "application/json;charset=UTF-8",
		// "Content-Length":  "64",
		// "Host":            "prd.eu-ccapi.kia.com:8080",
		// "Connection":      "close",
		// "Accept-Encoding": "gzip, deflate",
		"User-Agent": "okhttp/3.10.0",
	}

	var token struct {
		ControlToken string `json:"controlToken"`
	}

	req, err := v.jsonRequest(http.MethodPut, kiaUrlSendPIN, headers, data)
	if err == nil {
		_, err = v.RequestJSON(req, &token)
	}

	auth.controlToken = ""
	if err == nil {
		auth.controlToken = "Bearer " + token.ControlToken
		auth.validUntil = time.Now().Add(time.Minute * 10)
	}

	return err
}

func (v *Kia) getStatus(ad KiaAuth) (float64, error) {
	headers := map[string]string{
		"Authorization":  ad.controlToken,
		"ccsp-device-id": ad.deviceId,
		"Content-Type":   "application/json",
	}

	var kr kiaBatteryResponse
	req, err := v.request(http.MethodGet, kiaUrlGetStatus+ad.vehicleID+"/status", headers, nil)
	if err == nil {
		_, err = v.RequestJSON(req, &kr)
	}

	return kr.ResMsg.EvStatus.BatteryStatus, err
}

func (v *Kia) connectToKiaServer() (err error) {
	v.Log.DEBUG.Println("connecting to Kia server")
	var kd KiaData
	var ad KiaAuth

	ad.deviceId, err = v.getDeviceID()
	if err == nil {
		time.Sleep(1 * time.Second)
		err = v.getCookies(&kd)
	}

	// if err = v.setLanguage(&kd); err != nil {
	// 	return errors.New("could not set language to en")
	// }
	// time.Sleep(1 * time.Second)

	if err == nil {
		time.Sleep(1 * time.Second)
		err = v.login(&kd)
	}

	if err == nil {
		time.Sleep(1 * time.Second)
		err = v.getToken(&kd)
	}

	if err == nil {
		time.Sleep(1 * time.Second)
		ad.vehicleID, err = v.getVehicles(&kd, ad.deviceId)
	}

	if err == nil {
		time.Sleep(1 * time.Second)
		err = v.prewakeup(&kd, ad.deviceId, ad.vehicleID)
	}

	if err == nil {
		time.Sleep(1 * time.Second)
		if err = v.sendPIN(&ad, kd); err == nil {
			v.auth = ad
		}
	}

	return err
}

// chargeState implements the Vehicle.ChargeState interface
func (v *Kia) chargeState() (float64, error) {
	soc, err := v.getStatus(v.auth)

	// TODO: recognize AUTH errors
	if err != nil {
		if err = v.connectToKiaServer(); err == nil {
			soc, err = v.getStatus(v.auth)
		}
	}

	return soc, err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Kia) ChargeState() (float64, error) {
	return v.chargeG()
}

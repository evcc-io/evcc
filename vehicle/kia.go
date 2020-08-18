package vehicle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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

type kiaErrorResponse struct {
	Error       string
	Description string `json:"error_description"`
}

// Kia is an api.Vehicle implementation with configurable getters and setters.
type Kia struct {
	*embed
	*util.HTTPHelper
	user         string
	password     string
	pin          int16
	cookieJar    *cookiejar.Jar
	accCode      string
	accToken     string
	deviceID     string
	vehicleID    string
	controlToken string
	chargeG      func() (float64, error)
}

type kiaBatteryResponse struct {
	ResMsg struct {
		EvStatus struct {
			BatteryStatus float64 `json:"batteryStatus"`
		} `json:"evStatus"`
	} `json:"resMsg"`
}

// NewKiaFromConfig creates a new Vehicle
func NewKiaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		PIN                 int16
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

// the following functions are implemented based on https://openwb.de/forum/viewtopic.php?f=5&t=1215&start=10#p11877

func (v *Kia) getDeviceID() error {
	uniId, _ := uuid.NewUUID()

	data := map[string]interface{}{
		"pushRegId": "1",
		"pushType":  "GCM",
		"uuid":      uniId.String(),
	}

	dataj, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, kiaUrlDeviceID, bytes.NewReader(dataj))
	if err != nil {
		return err
	}
	for k, v := range map[string]string{
		"ccsp-service-id": "fdc85c00-0a2f-4c64-bcb4-2cfb1500730a",
		"Content-type":    "application/json;charset=UTF-8",
		"Content-Length":  "80",
		"Host":            "prd.eu-ccapi.kia.com:8080",
		"Connection":      "close",
		"Accept-Encoding": "gzip, deflate",
		"User-Agent":      "okhttp/3.10.0",
	} {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var bbody map[string]interface{}
	json.Unmarshal([]byte(body), &bbody)

	v.deviceID = fmt.Sprint(bbody["resMsg"].(map[string]interface{})["deviceId"])

	return nil

}

func (v *Kia) getCookies() error {
	v.cookieJar, _ = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})

	client := &http.Client{Jar: v.cookieJar}
	req, err := http.NewRequest(http.MethodGet, kiaUrlCookies, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (v *Kia) setLanguage() error {
	data := map[string]interface{}{
		"lang": "en",
	}

	dataj, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, kiaUrlLang, bytes.NewReader(dataj))
	if err != nil {
		return err
	}
	for k, v := range map[string]string{
		"Content-type": "application/json",
	} {
		req.Header.Set(k, v)
	}

	client := &http.Client{Jar: v.cookieJar}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (v *Kia) login() error {
	data := map[string]interface{}{
		"email":    v.user,
		"password": v.password,
	}

	dataj, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, kiaUrlLogin, bytes.NewReader(dataj))
	for k, v := range map[string]string{
		"Content-type": "application/json",
	} {
		req.Header.Set(k, v)
	}

	client := &http.Client{Jar: v.cookieJar}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var bbody map[string]interface{}
	json.Unmarshal([]byte(body), &bbody)

	redirUrl := fmt.Sprint(bbody["redirectUrl"])
	parsed, err := url.Parse(redirUrl)
	if err != nil {
		return err
	}
	quer := parsed.Query()
	v.accCode = quer.Get("code")

	return nil
}

func (v *Kia) getToken() error {
	data := "grant_type=authorization_code&redirect_uri=https%3A%2F%2Fprd.eu-ccapi.kia.com%3A8080%2Fapi%2Fv1%2Fuser%2Foauth2%2Fredirect&code="
	data = data + v.accCode
	req, err := http.NewRequest(http.MethodPost, kiaUrlAccessToken, bytes.NewReader([]byte(data)))
	if err != nil {
		return err
	}
	for k, v := range map[string]string{
		"Authorization":   "Basic ZmRjODVjMDAtMGEyZi00YzY0LWJjYjQtMmNmYjE1MDA3MzBhOnNlY3JldA==",
		"Content-type":    "application/x-www-form-urlencoded",
		"Content-Length":  "150",
		"Host":            "prd.eu-ccapi.kia.com:8080",
		"Connection":      "close",
		"Accept-Encoding": "gzip, deflate",
		"User-Agent":      "okhttp/3.10.0",
	} {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var bbody map[string]interface{}
	json.Unmarshal([]byte(body), &bbody)

	v.accToken = fmt.Sprintf("%s %s", bbody["token_type"], bbody["access_token"])

	return nil
}

func (v *Kia) getVehicles() error {
	req, err := http.NewRequest(http.MethodGet, kiaUrlVehicles, nil)
	if err != nil {
		return err
	}
	for k, v := range map[string]string{
		"Authorization":       v.accToken,
		"ccsp-device-id":      v.deviceID,
		"ccsp-application-id": "693a33fa-c117-43f2-ae3b-61a02d24f417",
		"offset":              "1",
		"Host":                "prd.eu-ccapi.kia.com:8080",
		"Connection":          "close",
		"Accept-Encoding":     "gzip, deflate",
		"User-Agent":          "okhttp/3.10.0",
	} {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var bbody map[string]interface{}
	json.Unmarshal([]byte(body), &bbody)

	kvid := bbody["resMsg"].(map[string]interface{})["vehicles"]
	kvid1 := kvid.([]interface{})[0]

	v.vehicleID = fmt.Sprint(kvid1.(map[string]interface{})["vehicleId"])

	return nil
}

func (v *Kia) prewakeup() error {
	uri := kiaUrlPreWakeup + v.vehicleID + "/control/engine"
	data := map[string]interface{}{
		"action":   "prewakeup",
		"deviceId": v.deviceID,
	}

	dataj, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewReader(dataj))
	for k, v := range map[string]string{
		"Authorization":       v.accToken,
		"ccsp-device-id":      v.deviceID,
		"ccsp-application-id": "693a33fa-c117-43f2-ae3b-61a02d24f417",
		"offset":              "1",
		"Content-Type":        "application/json;charset=UTF-8",
		"Content-Length":      "72",
		"Host":                "prd.eu-ccapi.kia.com:8080",
		"Connection":          "close",
		"Accept-Encoding":     "gzip, deflate",
		"User-Agent":          "okhttp/3.10.0",
	} {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (v *Kia) sendPIN() error {
	data := map[string]interface{}{
		"deviceId": v.deviceID,
		"pin":      string(v.pin),
	}

	dataj, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, kiaUrlSendPIN, bytes.NewReader(dataj))
	if err != nil {
		return err
	}
	for k, v := range map[string]string{
		"Authorization":   v.accToken,
		"Content-type":    "application/json;charset=UTF-8",
		"Content-Length":  "64",
		"Host":            "prd.eu-ccapi.kia.com:8080",
		"Connection":      "close",
		"Accept-Encoding": "gzip, deflate",
		"User-Agent":      "okhttp/3.10.0",
	} {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var bbody map[string]interface{}
	json.Unmarshal([]byte(body), &bbody)

	v.controlToken = "Bearer " + fmt.Sprint(bbody["controlToken"])

	return nil
}

func (v *Kia) getStatus() (float64, error) {
	uri := kiaUrlGetStatus + v.vehicleID + "/status"
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return 0.0, err
	}
	for k, v := range map[string]string{
		"Authorization":  v.controlToken,
		"ccsp-device-id": v.deviceID,
		"Content-Type":   "application/json",
	} {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0.0, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var kr kiaBatteryResponse
	json.Unmarshal([]byte(body), &kr)
	SoC := kr.ResMsg.EvStatus.BatteryStatus

	return SoC, nil
}

// now we have all the needed functions to read Kia SoC
// chargeState implements the Vehicle.ChargeState interface
func (v *Kia) chargeState() (float64, error) {
	err := v.getDeviceID()
	if err != nil {
		return 0, errors.New("could not obtain deviceID")
	}

	err = v.getCookies()
	if err != nil {
		return 0, errors.New("could not obtain cookies")
	}
	err = v.setLanguage()
	if err != nil {
		return 0, errors.New("could not set language to en")
	}
	err = v.login()
	if err != nil {
		return 0, errors.New("could not login")
	}
	err = v.getToken()
	if err != nil {
		return 0, errors.New("could not obtain token")
	}
	err = v.getVehicles()
	if err != nil {
		return 0, errors.New("could not obtain vehicleID")
	}
	err = v.prewakeup()
	if err != nil {
		return 0, errors.New("could not trigger prewakeup")
	}
	err = v.sendPIN()
	if err != nil {
		return 0, errors.New("could not send pin")
	}
	soc, errf := v.getStatus()
	if errf != nil {
		return 0, errors.New("could not get soc")
	}

	return soc, nil
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Kia) ChargeState() (float64, error) {
	return v.chargeG()
}

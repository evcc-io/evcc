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

// Kia is an api.Vehicle implementation with configurable getters and setters.
type Kia struct {
	*embed
	*util.HTTPHelper
	user     string
	password string
	pin      string
	chargeG  func() (float64, error)
}

type KiaData struct {
	cookieJar *cookiejar.Jar
	accCode   string
	accToken  string
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

// Credits to https://openwb.de/forum/viewtopic.php?f=5&t=1215&start=10#p11877
func (v *Kia) getDeviceID() (string, error) {
	uniId, _ := uuid.NewUUID()

	data := map[string]interface{}{
		"pushRegId": "1",
		"pushType":  "GCM",
		"uuid":      uniId.String(),
	}

	dataj, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, kiaUrlDeviceID, bytes.NewReader(dataj))
	if err != nil {
		return "", err
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

	var did deviceIdResponse
	if _, err := v.RequestJSON(req, &did); err != nil {
		return "", err
	}

	devid := fmt.Sprintf(did.ResMsg.DeviceId)

	return devid, nil

}

func (v *Kia) getCookies(kd *KiaData) error {
	kd.cookieJar, _ = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})

	client := &http.Client{Jar: kd.cookieJar}
	req, err := http.NewRequest(http.MethodGet, kiaUrlCookies, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func (v *Kia) setLanguage(kd *KiaData) error {
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

	client := &http.Client{Jar: kd.cookieJar}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (v *Kia) login(kd *KiaData) error {
	data := map[string]interface{}{
		"email":    v.user,
		"password": v.password,
	}

	dataj, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, kiaUrlLogin, bytes.NewReader(dataj))
	if err != nil {
		return err
	}
	for k, v := range map[string]string{
		"Content-type": "application/json",
	} {
		req.Header.Set(k, v)
	}

	client := &http.Client{Jar: kd.cookieJar}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var bbody map[string]interface{}
	err = json.Unmarshal([]byte(body), &bbody)
	if err != nil {
		return err
	}
	redirUrl := fmt.Sprint(bbody["redirectUrl"])
	parsed, err := url.Parse(redirUrl)
	if err != nil {
		return err
	}
	quer := parsed.Query()
	kd.accCode = quer.Get("code")

	return nil
}

func (v *Kia) getToken(kd *KiaData) error {
	data := "grant_type=authorization_code&redirect_uri=https%3A%2F%2Fprd.eu-ccapi.kia.com%3A8080%2Fapi%2Fv1%2Fuser%2Foauth2%2Fredirect&code="
	data = data + kd.accCode
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var bbody map[string]interface{}
	err = json.Unmarshal([]byte(body), &bbody)
	if err != nil {
		return err
	}

	kd.accToken = fmt.Sprintf("%s %s", bbody["token_type"], bbody["access_token"])

	return nil
}

func (v *Kia) getVehicles(kd *KiaData, did string) (string, error) {

	req, err := http.NewRequest(http.MethodGet, kiaUrlVehicles, nil)
	if err != nil {
		return "", err
	}
	for k, v := range map[string]string{
		"Authorization":       kd.accToken,
		"ccsp-device-id":      did,
		"ccsp-application-id": "693a33fa-c117-43f2-ae3b-61a02d24f417",
		"offset":              "1",
		"Host":                "prd.eu-ccapi.kia.com:8080",
		"Connection":          "close",
		"Accept-Encoding":     "gzip, deflate",
		"User-Agent":          "okhttp/3.10.0",
	} {
		req.Header.Set(k, v)
	}

	body, err := v.Request(req)
	if err != nil {
		return "", err
	}

	var vid vehicleIdResponse
	err = json.Unmarshal([]byte(body), &vid)
	if err != nil {
		return "", err
	}

	vehid := fmt.Sprint(vid.ResMsg.Vehicles[0].VehicleId)

	return vehid, nil
}

func (v *Kia) prewakeup(kd *KiaData, did, vid string) error {
	uri := kiaUrlPreWakeup + vid + "/control/engine"
	data := map[string]interface{}{
		"action":   "prewakeup",
		"deviceId": did,
	}

	dataj, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewReader(dataj))
	if err != nil {
		return err
	}
	for k, v := range map[string]string{
		"Authorization":       kd.accToken,
		"ccsp-device-id":      did,
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
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (v *Kia) sendPIN(auth *KiaAuth, kd *KiaData) error {

	auth.controlToken = ""

	data := map[string]interface{}{
		"deviceId": auth.deviceId,
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
		"Authorization":   kd.accToken,
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var bbody map[string]interface{}
	err = json.Unmarshal([]byte(body), &bbody)
	if err != nil {
		return err
	}

	auth.controlToken = "Bearer " + fmt.Sprint(bbody["controlToken"])
	auth.validUntil = time.Now().Add(time.Minute * 10)

	return nil
}

func (v *Kia) getStatus(ad KiaAuth) (float64, error) {
	uri := kiaUrlGetStatus + ad.vehicleID + "/status"
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return 0.0, err
	}
	for k, v := range map[string]string{
		"Authorization":  ad.controlToken,
		"ccsp-device-id": ad.deviceId,
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var kr kiaBatteryResponse
	err = json.Unmarshal([]byte(body), &kr)
	if err != nil {
		return 0, err
	}
	stateOfCharge := kr.ResMsg.EvStatus.BatteryStatus

	return stateOfCharge, nil
}

func (v *Kia) connectToKiaServer() (KiaAuth, error) {
	v.Log.DEBUG.Println("connecting to Kia server")
	var kd KiaData
	var ad KiaAuth
	var err error

	if ad.deviceId, err = v.getDeviceID(); err != nil {
		return ad, errors.New("could not obtain deviceID")
	}
	time.Sleep(1 * time.Second)

	if err = v.getCookies(&kd); err != nil {
		return ad, errors.New("could not obtain cookies")
	}
	time.Sleep(1 * time.Second)

	if err = v.setLanguage(&kd); err != nil {
		return ad, errors.New("could not set language to en")
	}
	time.Sleep(1 * time.Second)

	if err = v.login(&kd); err != nil {
		return ad, errors.New("could not login")
	}
	time.Sleep(1 * time.Second)

	if err = v.getToken(&kd); err != nil {
		return ad, errors.New("could not obtain token")
	}
	time.Sleep(1 * time.Second)

	if ad.vehicleID, err = v.getVehicles(&kd, ad.deviceId); err != nil {
		return ad, errors.New("could not obtain vehicleID")
	}
	time.Sleep(1 * time.Second)

	if err = v.prewakeup(&kd, ad.deviceId, ad.vehicleID); err != nil {
		return ad, errors.New("could not trigger prewakeup")
	}
	time.Sleep(1 * time.Second)

	if err = v.sendPIN(&ad, &kd); err != nil {
		return ad, errors.New("could not send pin")
	}
	time.Sleep(1 * time.Second)
	v.Log.DEBUG.Println("auth received from Kia server")
	return ad, nil
}

// now we have all the needed functions to read Kia SoC
// chargeState implements the Vehicle.ChargeState interface
func (v *Kia) chargeState() (float64, error) {
	//fmt.Println("SoC Abfrage gestartet")
	var auth KiaAuth
	var soc float64
	var err error
	const maxretry = 5
	var i int = 0

	// retry login max 5 times
	for ok := true; ok; ok = !(err == nil || i > maxretry) {
		auth, err = v.connectToKiaServer()
		i++

		if i > 1 {
			time.Sleep(time.Second * 1)
			v.Log.DEBUG.Println("Kia login retry")
		}
	}
	if err != nil || i > maxretry {
		return 0, errors.New("could not connect to Kia server")
	}

	i = 0
	for ok := true; ok; ok = !((err == nil && soc > 0) || i > maxretry) {
		soc, err = v.getStatus(auth)
		i++

		if i > 1 {
			time.Sleep(time.Second * 1)
			v.Log.DEBUG.Println("Kia getStatus retry")
		}
	}

	if err != nil || i > maxretry {
		return 0, errors.New("could not get soc")
	}

	v.Log.DEBUG.Println("SoC received: ", soc)

	return soc, nil
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Kia) ChargeState() (float64, error) {
	return v.chargeG()
}

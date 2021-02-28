package bluelink

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
	"github.com/google/uuid"
	"github.com/imdario/mergo"
	"golang.org/x/net/publicsuffix"
)

const (
	resOK       = "S"
	resAuthFail = "F"
)

var (
	errAuthFail = errors.New("authorization failed")

	defaults = Config{
		DeviceID:    "/api/v1/spa/notifications/register",
		Lang:        "/api/v1/user/language",
		Login:       "/api/v1/user/signin",
		AccessToken: "/api/v1/user/oauth2/token",
		Vehicles:    "/api/v1/spa/vehicles",
		Status:      "/api/v1/spa/vehicles/%s/status",
	}
)

// Config is the bluelink API configuration
type Config struct {
	URI               string
	TokenAuth         string
	CCSPServiceID     string
	CCSPApplicationID string
	DeviceID          string
	Lang              string
	Login             string
	AccessToken       string
	Vehicles          string
	Status            string
}

// API implements the Kia/Hyundai bluelink api.
// Based on https://github.com/Hacksore/bluelinky.
type API struct {
	*request.Helper
	log      *util.Logger
	user     string
	password string
	apiG     func() (interface{}, error)
	config   Config
	auth     Auth
}

// Auth bundles miscellaneous authorization data
type Auth struct {
	accToken  string
	deviceID  string
	vehicleID string
}

type response struct {
	timestamp time.Time // add missing timestamp
	RetCode   string
	ResMsg    struct {
		DeviceID string
		EvStatus struct {
			BatteryStatus float64
			RemainTime2   struct {
				Atc struct {
					Value, Unit int
				}
			}
			DrvDistance []drvDistance
		}
		Vehicles []struct {
			VehicleID string
		}
	}
}

type drvDistance struct {
	RangeByFuel struct {
		EvModeRange struct {
			Value int
		}
	}
}

// New creates a new BlueLink API
func New(log *util.Logger, user, password string, cache time.Duration, config Config) (*API, error) {
	if err := mergo.Merge(&config, defaults); err != nil {
		return nil, err
	}

	v := &API{
		log:      log,
		Helper:   request.NewHelper(log),
		config:   config,
		user:     user,
		password: password,
	}

	// api is unbelievably slow when retrieving status
	v.Helper.Client.Timeout = 120 * time.Second

	v.apiG = provider.NewCached(v.statusAPI, cache).InterfaceGetter()

	return v, nil
}

// Credits to https://openwb.de/forum/viewtopic.php?f=5&t=1215&start=10#p11877

func (v *API) getDeviceID() (string, error) {
	uniID, _ := uuid.NewUUID()
	data := map[string]interface{}{
		"pushRegId": "1",
		"pushType":  "GCM",
		"uuid":      uniID.String(),
	}

	headers := map[string]string{
		"ccsp-service-id": v.config.CCSPServiceID,
		"Content-type":    "application/json;charset=UTF-8",
		"User-Agent":      "okhttp/3.10.0",
	}

	var resp response
	req, err := request.New(http.MethodPost, v.config.URI+v.config.DeviceID, request.MarshalJSON(data), headers)
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	return resp.ResMsg.DeviceID, err
}

func (v *API) getCookies() (cookieClient *request.Helper, err error) {
	cookieClient = request.NewHelper(v.log)
	cookieClient.Client.Jar, err = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	if err == nil {
		uri := fmt.Sprintf(
			"%s/api/v1/user/oauth2/authorize?response_type=code&state=test&client_id=%s&redirect_uri=%s/api/v1/user/oauth2/redirect",
			v.config.URI,
			v.config.CCSPServiceID,
			v.config.URI,
		)

		var resp *http.Response
		if resp, err = cookieClient.Get(uri); err == nil {
			resp.Body.Close()
		}
	}

	return cookieClient, err
}

func (v *API) setLanguage(cookieClient *request.Helper) error {
	data := map[string]interface{}{
		"lang": "en",
	}

	req, err := request.New(http.MethodPost, v.config.URI+v.config.Lang, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		var resp *http.Response
		if resp, err = cookieClient.Do(req); err == nil {
			resp.Body.Close()
		}
	}

	return err
}

func (v *API) login(cookieClient *request.Helper) (string, error) {
	data := map[string]interface{}{
		"email":    v.user,
		"password": v.password,
	}

	req, err := request.New(http.MethodPost, v.config.URI+v.config.Login, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return "", err
	}

	var redirect struct {
		RedirectURL string `json:"redirectUrl"`
	}

	var accCode string
	if err = cookieClient.DoJSON(req, &redirect); err == nil {
		if parsed, err := url.Parse(redirect.RedirectURL); err == nil {
			accCode = parsed.Query().Get("code")
		}
	}

	return accCode, err
}

func (v *API) getToken(accCode string) (string, error) {
	headers := map[string]string{
		"Authorization": "Basic " + v.config.TokenAuth,
		"Content-type":  "application/x-www-form-urlencoded",
		"User-Agent":    "okhttp/3.10.0",
	}

	data := url.Values(map[string][]string{
		"grant_type":   {"authorization_code"},
		"redirect_uri": {v.config.URI + "/api/v1/user/oauth2/redirect"},
		"code":         {accCode},
	})

	req, err := request.New(http.MethodPost, v.config.URI+v.config.AccessToken, strings.NewReader(data.Encode()), headers)
	if err != nil {
		return "", err
	}

	var tokens struct {
		TokenType   string `json:"token_type"`
		AccessToken string `json:"access_token"`
	}

	var accToken string
	if err = v.DoJSON(req, &tokens); err == nil {
		accToken = fmt.Sprintf("%s %s", tokens.TokenType, tokens.AccessToken)
	}

	return accToken, err
}

func (v *API) getVehicles(accToken, did string) (string, error) {
	headers := map[string]string{
		"Authorization":       accToken,
		"ccsp-device-id":      did,
		"ccsp-application-id": v.config.CCSPApplicationID,
		"offset":              "1",
		"User-Agent":          "okhttp/3.10.0",
	}

	req, err := request.New(http.MethodGet, v.config.URI+v.config.Vehicles, nil, headers)
	if err == nil {
		var resp response
		if err = v.DoJSON(req, &resp); err == nil {
			if len(resp.ResMsg.Vehicles) == 1 {
				return resp.ResMsg.Vehicles[0].VehicleID, nil
			}

			err = errors.New("couldn't find vehicle")
		}
	}

	return "", err
}

func (v *API) authFlow() (err error) {
	v.auth.deviceID, err = v.getDeviceID()

	var cookieClient *request.Helper
	if err == nil {
		cookieClient, err = v.getCookies()
	}

	if err == nil {
		err = v.setLanguage(cookieClient)
	}

	var accCode string
	if err == nil {
		accCode, err = v.login(cookieClient)
	}

	if err == nil {
		v.auth.accToken, err = v.getToken(accCode)
	}

	if err == nil {
		v.auth.vehicleID, err = v.getVehicles(v.auth.accToken, v.auth.deviceID)
	}

	return err
}

func (v *API) getStatus() (response, error) {
	var resp response

	if v.auth.accToken == "" {
		return resp, errAuthFail
	}

	headers := map[string]string{
		"Authorization":       v.auth.accToken,
		"ccsp-device-id":      v.auth.deviceID,
		"ccsp-application-id": v.config.CCSPApplicationID,
		"offset":              "1",
		"User-Agent":          "okhttp/3.10.0",
	}

	uri := fmt.Sprintf(v.config.URI+v.config.Status, v.auth.vehicleID)
	req, err := request.New(http.MethodGet, uri, nil, headers)
	if err == nil {
		err = v.DoJSON(req, &resp)

		if err != nil {
			// handle http 401, 403
			if se, ok := err.(request.StatusError); ok && se.HasStatus(http.StatusUnauthorized, http.StatusForbidden) {
				err = errAuthFail
			}
		}

		if err == nil && resp.RetCode != resOK {
			err = errors.New("unexpected response")
			if resp.RetCode == resAuthFail {
				err = errAuthFail
			}
		}
	}

	return resp, err
}

// status retrieves the bluelink status response
func (v *API) statusAPI() (interface{}, error) {
	res, err := v.getStatus()

	if err != nil && errors.Is(err, errAuthFail) {
		if err = v.authFlow(); err == nil {
			res, err = v.getStatus()
		}
	}

	// add local timestamp for FinishTime
	res.timestamp = time.Now()

	return res, err
}

// SoC implements the api.Vehicle interface
func (v *API) SoC() (float64, error) {
	res, err := v.apiG()

	if res, ok := res.(response); err == nil && ok {
		return float64(res.ResMsg.EvStatus.BatteryStatus), nil
	}

	return 0, err
}

// FinishTime implements the api.VehicleFinishTimer interface
func (v *API) FinishTime() (time.Time, error) {
	res, err := v.apiG()

	if res, ok := res.(response); err == nil && ok {
		remaining := res.ResMsg.EvStatus.RemainTime2.Atc.Value

		if remaining == 0 {
			return time.Time{}, api.ErrNotAvailable
		}

		return res.timestamp.Add(time.Duration(remaining) * time.Minute), nil
	}

	return time.Time{}, err
}

// Range implements the api.VehicleRange interface
func (v *API) Range() (int64, error) {
	res, err := v.apiG()

	if res, ok := res.(response); err == nil && ok {
		if dist := res.ResMsg.EvStatus.DrvDistance; len(dist) == 1 {
			return int64(dist[0].RangeByFuel.EvModeRange.Value), nil
		}

		return 0, api.ErrNotAvailable
	}

	return 0, err
}

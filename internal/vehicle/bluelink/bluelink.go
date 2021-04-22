package bluelink

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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
		DeviceID:        "/api/v1/spa/notifications/register",
		IntegrationInfo: "/api/v1/user/integrationinfo",
		SilentSignin:    "/api/v1/user/silentsignin",
		Lang:            "/api/v1/user/language",
		Login:           "/api/v1/user/signin",
		AccessToken:     "/api/v1/user/oauth2/token",
		Vehicles:        "/api/v1/spa/vehicles",
		Status:          "/api/v1/spa/vehicles/%s/status",
	}
)

// Config is the bluelink API configuration
type Config struct {
	URI               string
	BrandAuthUrl      string // v2
	IntegrationInfo   string // v2
	SilentSignin      string // v2
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
	Vehicle  Vehicle
}

// Auth bundles miscellaneous authorization data
type Auth struct {
	accToken string
	deviceID string
}

type Vehicle struct {
	Vin, VehicleName, VehicleID string
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
			DrvDistance []DrivingDistance
		}
		Vehicles []Vehicle
	}
}

type DrivingDistance struct {
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

func (v *API) stamp() string {
	return stamps.New(v.config.CCSPApplicationID)
}

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
		"Stamp":           v.stamp(),
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

func (v *API) brandLogin(cookieClient *request.Helper) (string, error) {
	req, err := request.New(http.MethodGet, v.config.URI+v.config.IntegrationInfo, nil, request.JSONEncoding)

	var info struct {
		UserId    string `json:"userId"`
		ServiceId string `json:"serviceId"`
	}

	if err == nil {
		err = cookieClient.DoJSON(req, &info)
	}

	var action string
	var resp *http.Response

	if err == nil {
		uri := fmt.Sprintf(v.config.BrandAuthUrl, v.config.URI, "en", info.ServiceId, info.UserId)

		req, err = request.New(http.MethodGet, uri, nil)
		if err == nil {
			if resp, err = cookieClient.Do(req); err == nil {
				defer resp.Body.Close()

				var doc *goquery.Document
				if doc, err = goquery.NewDocumentFromReader(resp.Body); err == nil {
					err = errors.New("form not found")

					if form := doc.Find("form"); form != nil && form.Length() == 1 {
						var ok bool
						if action, ok = form.Attr("action"); ok {
							err = nil
						}
					}
				}
			}
		}
	}

	if err == nil {
		data := url.Values{
			"username":     []string{v.user},
			"password":     []string{v.password},
			"credentialId": []string{""},
			"rememberMe":   []string{"on"},
		}

		req, err = request.New(http.MethodPost, action, strings.NewReader(data.Encode()), request.URLEncoding)
		if err == nil {
			cookieClient.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse } // don't follow redirects
			if resp, err = cookieClient.Do(req); err == nil {
				defer resp.Body.Close()

				// need 302
				if resp.StatusCode != http.StatusFound {
					err = errors.New("missing redirect")

					if doc, err2 := goquery.NewDocumentFromReader(resp.Body); err2 == nil {
						if span := doc.Find("span[class=kc-feedback-text]"); span != nil && span.Length() == 1 {
							err = errors.New(span.Text())
						}
					}
				}
			}

			cookieClient.CheckRedirect = nil
		}
	}

	var userId string
	if err == nil {
		resp, err = cookieClient.Get(resp.Header.Get("Location"))
		if err == nil {
			defer resp.Body.Close()

			userId = resp.Request.URL.Query().Get("userId")
			if len(userId) == 0 {
				err = errors.New("usedId not found")
			}
		}
	}

	var code string
	if err == nil {
		data := map[string]string{
			"userId": userId,
		}

		req, err = request.New(http.MethodPost, v.config.URI+v.config.SilentSignin, request.MarshalJSON(data), request.JSONEncoding)
		if err == nil {
			req.Header.Set("ccsp-service-id", v.config.CCSPServiceID)
			req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 11_1 like Mac OS X) AppleWebKit/604.3.5 (KHTML, like Gecko) Version/11.0 Mobile/15B92 Safari/604.1")

			cookieClient.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse } // don't follow redirects
			var res struct {
				RedirectUrl string `json:"redirectUrl"`
			}
			if err = cookieClient.DoJSON(req, &res); err == nil {
				fmt.Println(res)
				var uri *url.URL
				if uri, err = url.Parse(res.RedirectUrl); err == nil {
					if code = uri.Query().Get("code"); len(code) == 0 {
						err = errors.New("code not found")
					}
				}
			}
		}
	}

	return code, err
}

func (v *API) bluelinkLogin(cookieClient *request.Helper) (string, error) {
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

func (v *API) Vehicles() ([]Vehicle, error) {
	if v.auth.accToken == "" {
		if err := v.authFlow(); err != nil {
			return nil, err
		}
	}

	headers := map[string]string{
		"Authorization":       v.auth.accToken,
		"ccsp-device-id":      v.auth.deviceID,
		"ccsp-application-id": v.config.CCSPApplicationID,
		"offset":              "1",
		"User-Agent":          "okhttp/3.10.0",
		"Stamp":               v.stamp(),
	}

	req, err := request.New(http.MethodGet, v.config.URI+v.config.Vehicles, nil, headers)

	var resp response
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	return resp.ResMsg.Vehicles, err
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
		// try new login first, then fallback
		if accCode, err = v.brandLogin(cookieClient); err != nil {
			accCode, err = v.bluelinkLogin(cookieClient)
		}

		if err != nil {
			err = fmt.Errorf("login failed: %w", err)
		}
	}

	if err == nil {
		v.auth.accToken, err = v.getToken(accCode)
	}

	if err != nil {
		err = fmt.Errorf("login failed: %w", err)
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
		"Stamp":               v.stamp(),
	}

	uri := fmt.Sprintf(v.config.URI+v.config.Status, v.Vehicle.VehicleID)
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

var _ api.Battery = (*API)(nil)

// SoC implements the api.Vehicle interface
func (v *API) SoC() (float64, error) {
	res, err := v.apiG()

	if res, ok := res.(response); err == nil && ok {
		return float64(res.ResMsg.EvStatus.BatteryStatus), nil
	}

	return 0, err
}

var _ api.VehicleFinishTimer = (*API)(nil)

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

var _ api.VehicleRange = (*API)(nil)

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

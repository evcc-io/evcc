package bluelink

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"
)

const (
	DeviceIdURL        = "/api/v1/spa/notifications/register"
	IntegrationInfoURL = "/api/v1/user/integrationinfo"
	SilentSigninURL    = "/api/v1/user/silentsignin"
	LanguageURL        = "/api/v1/user/language"
	LoginURL           = "/api/v1/user/signin"
	TokenURL           = "/api/v1/user/oauth2/token"
)

// Config is the bluelink API configuration
type Config struct {
	URI               string
	AuthClientID      string // v2
	BrandAuthUrl      string // v2
	BasicToken        string
	CCSPServiceID     string
	CCSPApplicationID string
	PushType          string
	Cfb               string
	LoginFormHost     string
	Brand             string
}

// Identity implements the Kia/Hyundai bluelink identity.
// Based on https://github.com/Hacksore/bluelinky.
type Identity struct {
	*request.Helper
	log      *util.Logger
	config   Config
	deviceID string
	oauth2.TokenSource
}

// NewIdentity creates BlueLink Identity
func NewIdentity(log *util.Logger, config Config) *Identity {
	v := &Identity{
		log:    log,
		Helper: request.NewHelper(log),
		config: config,
	}

	return v
}

func (v *Identity) getDeviceID() (string, error) {
	stamp, err := v.stamp()
	if err != nil {
		return "", err
	}

	uuid := uuid.NewString()
	data := map[string]interface{}{
		"pushRegId": lo.RandomString(64, []rune("0123456789ABCDEF")),
		"pushType":  v.config.PushType,
		"uuid":      uuid,
	}

	headers := map[string]string{
		"ccsp-service-id":     v.config.CCSPServiceID,
		"ccsp-application-id": v.config.CCSPApplicationID,
		"Content-type":        "application/json;charset=UTF-8",
		"User-Agent":          "okhttp/3.10.0",
		"Stamp":               stamp,
	}

	var res struct {
		RetCode string
		ResMsg  struct {
			DeviceID string
		}
	}

	req, err := request.New(http.MethodPost, v.config.URI+DeviceIdURL, request.MarshalJSON(data), headers)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	if res.ResMsg.DeviceID == "" {
		err = errors.New("deviceid not found")
	}

	return res.ResMsg.DeviceID, err
}

func (v *Identity) getCookies() (cookieClient *request.Helper, err error) {
	cookieClient = request.NewHelper(v.log)
	cookieClient.Client.Jar, _ = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	// TODO: check whether &lang= is necessary
	uri := fmt.Sprintf(
		"%s/api/v1/user/oauth2/authorize?response_type=code&state=test&client_id=%s&redirect_uri=%s/api/v1/user/oauth2/redirect",
		v.config.URI,
		v.config.CCSPServiceID,
		v.config.URI,
	)

	resp, err := cookieClient.Get(uri)
	if err == nil {
		resp.Body.Close()
	}

	return cookieClient, err
}

func (v *Identity) setLanguage(cookieClient *request.Helper, language string) error {
	data := map[string]interface{}{
		"lang": language,
	}

	req, err := request.New(http.MethodPost, v.config.URI+LanguageURL, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		var resp *http.Response
		if resp, err = cookieClient.Do(req); err == nil {
			resp.Body.Close()
		}
	}

	return err
}

func (v *Identity) brandLoginHyundaiEU(cookieClient *request.Helper, user, password string) (string, error) {
	req, err := request.New(http.MethodGet, v.config.URI+IntegrationInfoURL, nil, request.JSONEncoding)

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
		uri := fmt.Sprintf(v.config.BrandAuthUrl, v.config.AuthClientID, v.config.URI, "en", info.ServiceId, info.UserId)

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
			"username":     {user},
			"password":     {password},
			"credentialId": {""},
			"rememberMe":   {"on"},
		}

		req, err = request.New(http.MethodPost, action, strings.NewReader(data.Encode()), request.URLEncoding)
		if err == nil {
			cookieClient.CheckRedirect = request.DontFollow
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

	if err == nil {
		resp, err = cookieClient.Get(resp.Header.Get("Location"))
		if err == nil {
			defer resp.Body.Close()
		}
	}

	var code string
	if err == nil {
		data := map[string]string{
			"intUserId": "",
		}

		req, err = request.New(http.MethodPost, v.config.URI+SilentSigninURL, request.MarshalJSON(data), request.JSONEncoding)
		if err == nil {
			req.Header.Set("ccsp-service-id", v.config.CCSPServiceID)
			cookieClient.CheckRedirect = request.DontFollow

			var res struct {
				RedirectUrl string `json:"redirectUrl"`
			}

			if err = cookieClient.DoJSON(req, &res); err == nil {
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

/* Unused for now
func (v *Identity) brandLoginKiaEU(user, password string) (string, error) {
	cookieClient := request.NewHelper(v.log)
	cookieClient.Client.Jar, _ = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})

	headers := map[string]string{
		"content-type": "application/x-www-form-urlencoded",
		"User-Agent":   "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.166 Mobile Safari/535.19_CCS_APP_AOS",
		// "User-Agent":   "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.166 Mobile Safari/535.19",
	}

	data := url.Values{
		"client_id":         {"peukiaidm-online-sales"},
		"encryptedPassword": {"false"},
		"password":          {password},
		"redirect_uri":      {"https://www.kia.com/api/bin/oneid/login"},
		"state":             {"aHR0cHM6Ly93d3cua2lhLmNvbTo0NDMvZGUvP3ZlZD0yYWhVS0V3akI2ZFc3dDQtUEF4WFBSZkVESGNDQ0J4UVFnVTk2QkFnY0VBZyZfdG09MTc1NTg1NTY2ODE2Mg==_default"},
		"username":          {user},
		"remember_me":       {"false"},
	}

	req, _ := request.New(http.MethodPost, "https://idpconnect-eu.kia.com/auth/account/signin", strings.NewReader(data.Encode()), headers)

	if _, err := cookieClient.Do(req); err != nil {
		return "", err
	}

	v.deviceID, _ = v.getDeviceID()

	// get the connector_session_key
	uri := fmt.Sprintf(v.config.BrandAuthUrl, v.config.LoginFormHost, v.config.CCSPServiceID, v.config.URI, "en")
	headers = map[string]string{
		"ccsp-application-id": v.config.CCSPApplicationID,
		"ccsp-device-id":      v.deviceID,
		"ccsp-service-id":     v.config.CCSPServiceID,
		"User-Agent":          "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.166 Mobile Safari/535.19",
	}
	req, _ = request.New(http.MethodGet, uri, nil, headers)
	resp, err := cookieClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// get redirect URL from request
	nextUri := resp.Request.URL.Query().Get("next_uri")
	if nextUri == "" {
		return "", errors.New("empty redirect url on connector session key request")
	}

	// create a client that doesn't honor redirects so we receive the original response
	// no idea how to do that with the internal request.New(...) function
	sc := http.Client{
		Jar:       cookieClient.Client.Jar,
		Transport: request.NewTripper(v.log, http.DefaultTransport),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err = request.New(http.MethodGet, nextUri, nil, headers)
	if err != nil {
		return "", err
	}

	resp, err = sc.Do(req)
	if err != nil {
		return "", err
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return "", errors.New("missing location header")
	}

	locationUrl, err := url.Parse(location)
	if err != nil {
		return "", err
	}

	code := locationUrl.Query().Get("code")
	if code == "" {
		return "", errors.New("missing code")
	}

	return code, nil
}
*/

func (v *Identity) bluelinkLogin(cookieClient *request.Helper, user, password string) (string, error) {
	data := map[string]interface{}{
		"email":    user,
		"password": password,
	}

	req, err := request.New(http.MethodPost, v.config.URI+LoginURL, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return "", err
	}

	var res struct {
		RedirectURL string `json:"redirectUrl"`
		ErrCode     string `json:"errCode"`
		ErrMsg      string `json:"errMsg"`
	}

	var accCode string
	if err = cookieClient.DoJSON(req, &res); err == nil {
		if parsed, err := url.Parse(res.RedirectURL); err == nil {
			accCode = parsed.Query().Get("code")
		}
	} else if res.ErrCode != "" {
		err = fmt.Errorf("%w: %s (%s)", err, res.ErrMsg, res.ErrCode)
	}

	return accCode, err
}

func (v *Identity) exchangeCodeHyundaiEU(accCode string) (*oauth2.Token, error) {
	headers := map[string]string{
		"Authorization": "Basic " + v.config.BasicToken,
		"Content-type":  "application/x-www-form-urlencoded",
		"User-Agent":    "okhttp/3.10.0",
	}

	data := url.Values{
		"grant_type":   {"authorization_code"},
		"redirect_uri": {v.config.URI + "/api/v1/user/oauth2/redirect"},
		"code":         {accCode},
	}

	var token oauth2.Token

	req, _ := request.New(http.MethodPost, v.config.URI+TokenURL, strings.NewReader(data.Encode()), headers)
	err := v.DoJSON(req, &token)

	return util.TokenWithExpiry(&token), err
}

func (v *Identity) exchangeCodeKiaEURefreshToken(accCode string) (*oauth2.Token, error) {
	uri := v.config.LoginFormHost + "/auth/api/v2/user/oauth2/token"
	headers := map[string]string{
		"Content-type": "application/x-www-form-urlencoded",
		"User-Agent":   "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.166 Mobile Safari/535.19_CCS_APP_AOS",
	}
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {accCode},
		"client_id":     {v.config.CCSPServiceID},
		"client_secret": {"secret"},
	}

	var token oauth2.Token

	req, _ := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), headers)
	err := v.DoJSON(req, &token)

	// manually set the refresh token
	token.RefreshToken = accCode

	return util.TokenWithExpiry(&token), err
}

/* Unused for now
func (v *Identity) exchangeCodeKiaEU(accCode string) (*oauth2.Token, error) {
	uri := v.config.LoginFormHost + "/auth/api/v2/user/oauth2/token"
	headers := map[string]string{
		"Content-type": "application/x-www-form-urlencoded",
		"User-Agent":   "okhttp/3.10.0",
	}
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {accCode},
		"redirect_uri":  {v.config.URI + "/api/v1/user/oauth2/redirect"},
		"client_id":     {v.config.CCSPServiceID},
		"client_secret": {"secret"},
	}

	var token oauth2.Token

	req, _ := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), headers)
	err := v.DoJSON(req, &token)

	return util.TokenWithExpiry(&token), err
}
*/

// RefreshToken implements oauth.TokenRefresher
func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	var res oauth2.Token
	var err error
	var uri string
	var headers map[string]string
	var data url.Values
	switch v.config.Brand {
	case "hyundai":
		uri = v.config.URI + TokenURL
		headers = map[string]string{
			"Authorization": "Basic " + v.config.BasicToken,
			"Content-type":  "application/x-www-form-urlencoded",
			// "User-Agent":    "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.166 Mobile Safari/535.19_CCS_APP_AOS",
			"User-Agent": "okhttp/3.10.0",
		}

		data = url.Values{
			"grant_type":    {"refresh_token"},
			"redirect_uri":  {"https://www.getpostman.com/oauth2/callback"},
			"refresh_token": {token.RefreshToken},
		}

	case "kia":
		uri = v.config.LoginFormHost + "/auth/api/v2/user/oauth2/token"
		headers = map[string]string{
			"Content-type": "application/x-www-form-urlencoded",
			"User-Agent":   "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.166 Mobile Safari/535.19_CCS_APP_AOS",
		}
		data = url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {token.RefreshToken},
			"client_id":     {v.config.CCSPServiceID},
			"client_secret": {"secret"},
		}
	default:
		err = errors.New("Unsupported brand")
	}

	// request token only if we didn't run unto the default branch
	if err != nil {
		return nil, err
	}

	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), headers)
	if err != nil {
		return nil, err
	}

	err = v.DoJSON(req, &res)
	// carry over the old refresh token (if any and not populated already)
	if res.RefreshToken == "" && token.RefreshToken != "" {
		res.RefreshToken = token.RefreshToken
	}

	return util.TokenWithExpiry(&res), err
}

func (v *Identity) Login(user, password, language, brand string) (err error) {
	if user == "" || password == "" {
		return api.ErrMissingCredentials
	}
	var code string
	switch brand {
	case "kia":
		// the "password" now is the refresh token ...
		// code, err = v.brandLoginKiaEU(user, password)
		// if err == nil {
		var token *oauth2.Token
		token, err = v.exchangeCodeKiaEURefreshToken(password)
		if err == nil {
			v.TokenSource = oauth.RefreshTokenSource(token, v)
			v.deviceID, err = v.getDeviceID()
		}
		// }
	case "hyundai":
		v.deviceID, err = v.getDeviceID()

		var cookieClient *request.Helper
		if err == nil {
			cookieClient, err = v.getCookies()
		}

		if err == nil {
			err = v.setLanguage(cookieClient, language)
		}

		if err == nil {
			// try new login first, then fallback
			if code, err = v.brandLoginHyundaiEU(cookieClient, user, password); err != nil {
				code, err = v.bluelinkLogin(cookieClient, user, password)
			}
			if err == nil {
				var token *oauth2.Token
				if token, err = v.exchangeCodeHyundaiEU(code); err == nil {
					v.TokenSource = oauth.RefreshTokenSource(token, v)
				}
			}
		}
	default:
		err = fmt.Errorf("unknown brand (%s)", brand)
	}

	if err != nil {
		err = fmt.Errorf("login failed: %w", err)
	}

	return err
}

// Request decorates requests with authorization headers
func (v *Identity) Request(req *http.Request) error {
	// stamp, err := Stamps[v.config.CCSPApplicationID].Get()
	stamp, err := v.stamp()
	if err != nil {
		return err
	}

	token, err := v.Token()
	if err != nil {
		return err
	}

	for k, v := range map[string]string{
		"Authorization":       "Bearer " + token.AccessToken,
		"ccsp-device-id":      v.deviceID,
		"ccsp-application-id": v.config.CCSPApplicationID,
		"offset":              "1",
		"User-Agent":          "okhttp/3.10.0",
		"Stamp":               stamp,
	} {
		req.Header.Set(k, v)
	}

	return nil
}

// stamp creates a stamp locally according to https://github.com/Hyundai-Kia-Connect/hyundai_kia_connect_api/pull/371
func (v *Identity) stamp() (string, error) {
	cfb, err := base64.StdEncoding.DecodeString(v.config.Cfb)
	if err != nil {
		return "", err
	}

	raw := v.config.CCSPApplicationID + ":" + strconv.FormatInt(time.Now().UnixMilli(), 10)

	if len(cfb) != len(raw) {
		return "", fmt.Errorf("cfb and raw length not equal")
	}

	enc := make([]byte, 0, 50)
	for i := range cfb {
		enc = append(enc, cfb[i]^raw[i])
	}

	return base64.StdEncoding.EncodeToString(enc), nil
}

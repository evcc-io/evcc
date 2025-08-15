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

func (v *Identity) brandLoginKiaEU(cookieClient *request.Helper, user, password string) (string, error) {
	req, _ := request.New(http.MethodGet, v.config.URI+IntegrationInfoURL, nil, request.JSONEncoding)

	var info struct {
		UserId    string `json:"userId"`
		ServiceId string `json:"serviceId"`
	}

	if err := cookieClient.DoJSON(req, &info); err != nil {
		return "", err
	}

	var resp *http.Response

	// get the connector_session_key
	uri := fmt.Sprintf(v.config.BrandAuthUrl, v.config.LoginFormHost, v.config.AuthClientID, v.config.URI, "en")
	resp, err := cookieClient.Get(uri)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// get redirect URL from request
	nextUri := resp.Request.URL.Query().Get("next_uri")
	if nextUri == "" {
		return "", errors.New("empty redirect url on connector session key request")
	}

	nextVal, err := url.Parse(nextUri)
	if err != nil {
		return "", err
	}

	connectorSessionKey := nextVal.Query().Get("connector_session_key")
	if connectorSessionKey == "" {
		return "", errors.New("empty or non-existing connector session key")
	}

	// if we have the connectorSessionKey, go on and find the login code
	// build new request uri
	uri = fmt.Sprintf("%s%s", v.config.LoginFormHost, "/auth/account/signin")
	data := url.Values{
		"client_id":             {v.config.CCSPServiceID},
		"encryptedPassword":     {"false"},
		"orgHmgSid":             {""},
		"password":              {password},
		"redirect_uri":          {v.config.URI + "/api/v1/user/oauth2/redirect"},
		"state":                 {"ccsp"},
		"username":              {user},
		"remember_me":           {"false"},
		"connector_session_key": {connectorSessionKey},
		"_csrf":                 {""},
	}

	// create a client that doesn't honor redirects so we receive the original response
	// no idea how to do that with the internal request.New(...) function
	sc := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Origin":       v.config.LoginFormHost,
	})
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

// RefreshToken implements oauth.TokenRefresher
func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	headers := map[string]string{
		"Authorization": "Basic " + v.config.BasicToken,
		"Content-type":  "application/x-www-form-urlencoded",
		"User-Agent":    "okhttp/3.10.0",
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"redirect_uri":  {"https://www.getpostman.com/oauth2/callback"},
		"refresh_token": {token.RefreshToken},
	}

	req, err := request.New(http.MethodPost, v.config.URI+TokenURL, strings.NewReader(data.Encode()), headers)

	var res oauth2.Token
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return util.TokenWithExpiry(&res), err
}

func (v *Identity) Login(user, password, language, brand string) (err error) {
	if user == "" || password == "" {
		return api.ErrMissingCredentials
	}
	v.deviceID, err = v.getDeviceID()

	var cookieClient *request.Helper
	if err == nil {
		cookieClient, err = v.getCookies()
	}

	if err == nil {
		err = v.setLanguage(cookieClient, language)
	}

	var code string
	if err == nil {
		switch brand {
		case "kia":
			code, err = v.brandLoginKiaEU(cookieClient, user, password)
			if err == nil {
				var token *oauth2.Token
				if token, err = v.exchangeCodeKiaEU(code); err == nil {
					v.TokenSource = oauth.RefreshTokenSource(token, v)
				}
			}
		case "hyundai":
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
		default:
			err = fmt.Errorf("unknown brand (%s)", brand)
		}
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

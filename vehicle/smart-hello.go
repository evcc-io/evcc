package vehicle

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/smart/hello"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

// SmartHello is an api.Vehicle implementation for Smart Hello cars
type SmartHello struct {
	*embed
	*hello.Provider
}

func init() {
	registry.Add("smart-hello", NewSmartHelloFromConfig)
}

// NewSmartHelloFromConfig creates a new vehicle
func NewSmartHelloFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed          `mapstructure:",squash"`
		User, Password string
		VIN            string
		Expiry         time.Duration
		Cache          time.Duration
	}{
		Expiry: expiry,
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("smart-hello").Redact(cc.User, cc.Password, cc.VIN)

	v := &SmartHello{
		embed: &cc.embed,
	}

	//    const context = await this.requestClient({
	//   method: 'get',
	//   url: 'https://awsapi.future.smart.com/login-app/api/v1/authorize?uiLocales=de-DE&uiLocales=de-DE',
	//   headers: {
	//     'upgrade-insecure-requests': '1',
	//     'user-agent':
	//       'Mozilla/5.0 (Linux; Android 9; ANE-LX1 Build/HUAWEIANE-L21; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/118.0.0.0 Mobile Safari/537.36',
	//     accept:
	//       'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7',
	//     'x-requested-with': 'com.smart.hellosmart',
	//     'sec-fetch-site': 'none',
	//     'sec-fetch-mode': 'navigate',
	//     'sec-fetch-user': '?1',
	//     'sec-fetch-dest': 'document',
	//     'accept-language': 'de-DE,de;q=0.9,en-DE;q=0.8,en-US;q=0.7,en;q=0.6',
	//   },
	// }).then((res) => {
	//   this.log.debug(JSON.stringify(res.data));
	//   return qs.parse(res.request.path.split('?')[1]);
	// });

	// const loginResponse = await this.requestClient({
	//   method: 'post',
	//   maxBodyLength: Infinity,
	//   url: 'https://auth.smart.com/accounts.login',
	//   headers: {
	//     'user-agent':
	//       'Mozilla/5.0 (Linux; Android 9; ANE-LX1 Build/HUAWEIANE-L21; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/118.0.0.0 Mobile Safari/537.36',
	//     'content-type': 'application/x-www-form-urlencoded',
	//     accept: '*/*',
	//     origin: 'https://app.id.smart.com',
	//     'x-requested-with': 'com.smart.hellosmart',
	//     'sec-fetch-site': 'same-site',
	//     'sec-fetch-mode': 'cors',
	//     'sec-fetch-dest': 'empty',
	//     'accept-language': 'de-DE,de;q=0.9,en-DE;q=0.8,en-US;q=0.7,en;q=0.6',
	//     cookie:
	//       'gmid=gmid.ver4.AcbHPqUK5Q.xOaWPhRTb7gy-6-GUW6cxQVf_t7LhbmeabBNXqqqsT6dpLJLOWCGWZM07EkmfM4j.u2AMsCQ9ZsKc6ugOIoVwCgryB2KJNCnbBrlY6pq0W2Ww7sxSkUa9_WTPBIwAufhCQYkb7gA2eUbb6EIZjrl5mQ.sc3; ucid=hPzasmkDyTeHN0DinLRGvw; hasGmid=ver4; gig_bootstrap_3_L94eyQ-wvJhWm7Afp1oBhfTGXZArUfSHHW9p9Pncg513hZELXsxCfMWHrF8f5P5a=auth_ver4',
	//   },
	//   data: {
	//     loginID: this.config.username,
	//     password: this.config.password,
	//     sessionExpiration: '2592000',
	//     targetEnv: 'jssdk',
	//     include: 'profile,data,emails,subscriptions,preferences,',
	//     includeUserInfo: 'true',
	//     loginMode: 'standard',
	//     lang: 'de',
	//     riskContext:
	//       '{"b0":41187,"b1":[0,2,3,1],"b2":4,"b3":["-23|0.383","-81.33333587646484|0.236"],"b4":3,"b5":1,"b6":"Mozilla/5.0 (Linux; Android 9; ANE-LX1 Build/HUAWEIANE-L21; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/118.0.0.0 Mobile Safari/537.36","b7":[],"b8":"16:33:26","b9":-60,"b10":null,"b11":false,"b12":{"charging":true,"chargingTime":null,"dischargingTime":null,"level":0.58},"b13":[5,"360|760|24",false,true]}',
	//     source: 'showScreenSet',
	//     sdk: 'js_latest',
	//     authMode: 'cookie',
	//     pageURL: 'https://app.id.smart.com/login?gig_ui_locales=de-DE',
	//     sdkBuild: '15482',
	//     format: 'json',
	//   },
	// })

	// identity := mb.NewIdentity(log, hello.OAuth2Config)
	// err := identity.Login(cc.User, cc.Password)
	// if err != nil {
	// 	return v, fmt.Errorf("login failed: %w", err)
	// }

	// api := hello.NewAPI(log, identity)

	// cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	// if err == nil {
	// 	v.Provider = hello.NewProvider(log, api, cc.VIN, cc.Expiry, cc.Cache)
	// }

	client := request.NewHelper(log)

	uri := "https://awsapi.future.smart.com/login-app/api/v1/authorize?uiLocales=de-DE&uiLocales=de-DE"
	req, _ := request.New(http.MethodGet, uri, nil, map[string]string{
		"user-agent":       "Mozilla/5.0 (Linux; Android 9; ANE-LX1 Build/HUAWEIANE-L21; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/118.0.0.0 Mobile Safari/537.36",
		"x-requested-with": "com.smart.hellosmart",
	})
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	u := resp.Request.URL

	data := url.Values{
		"loginID":           {cc.User},
		"password":          {cc.Password},
		"sessionExpiration": {"2592000"},
		"targetEnv":         {"jssdk"},
		"include":           {"profile,data,emails,subscriptions,preferences,"},
		"includeUserInfo":   {"true"},
		"loginMode":         {"standard"},
		"lang":              {"de"},
		"APIKey":            {hello.ApiKey},
		"source":            {"showScreenSet"},
		"sdk":               {"js_latest"},
		"authMode":          {"cookie"},
		"pageURL":           {"https://app.id.smart.com/login?gig_ui_locales=de-DE"},
		"sdkBuild":          {"15482"},
		"format":            {"json"},
	}

	uri = "https://auth.smart.com/accounts.login"
	req, _ = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), map[string]string{
		"user-agent":       "Mozilla/5.0 (Linux; Android 9; ANE-LX1 Build/HUAWEIANE-L21; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/118.0.0.0 Mobile Safari/537.36",
		"x-requested-with": "com.smart.hellosmart",
		"content-type":     request.FormContent,
		"cookie":           "gmid=gmid.ver4.AcbHPqUK5Q.xOaWPhRTb7gy-6-GUW6cxQVf_t7LhbmeabBNXqqqsT6dpLJLOWCGWZM07EkmfM4j.u2AMsCQ9ZsKc6ugOIoVwCgryB2KJNCnbBrlY6pq0W2Ww7sxSkUa9_WTPBIwAufhCQYkb7gA2eUbb6EIZjrl5mQ.sc3; ucid=hPzasmkDyTeHN0DinLRGvw; hasGmid=ver4; gig_bootstrap_3_L94eyQ-wvJhWm7Afp1oBhfTGXZArUfSHHW9p9Pncg513hZELXsxCfMWHrF8f5P5a=auth_ver4",
	})

	var login struct {
		ErrorCode                  int
		ErrorDetails, ErrorMessage string
		SessionInfo                struct {
			LoginToken string `json:"login_token"`
			ExpiresIn  int    `json:"expires_in,string"`
		}
	}

	if err := client.DoJSON(req, &login); err != nil {
		return nil, err
	}
	if login.ErrorCode != 0 {
		return nil, fmt.Errorf("%s: %s", login.ErrorMessage, login.ErrorDetails)
	}
	defer resp.Body.Close()

	var param request.InterceptResult
	client.CheckRedirect, param = request.InterceptRedirect("access_token", true)

	fmt.Println(login)
	uri = fmt.Sprintf("https://auth.smart.com/oidc/op/v1.0/%s/authorize/continue?context=%s&login_token=%s", hello.ApiKey, u.Query().Get("context"), login.SessionInfo.LoginToken)
	req, _ = request.New(http.MethodGet, uri, nil, map[string]string{
		"user-agent":       "Mozilla/5.0 (Linux; Android 9; ANE-LX1 Build/HUAWEIANE-L21; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/118.0.0.0 Mobile Safari/537.36",
		"x-requested-with": "com.smart.hellosmart",
		"content-type":     request.FormContent,
		"cookie":           "gmid=gmid.ver4.AcbHPqUK5Q.xOaWPhRTb7gy-6-GUW6cxQVf_t7LhbmeabBNXqqqsT6dpLJLOWCGWZM07EkmfM4j.u2AMsCQ9ZsKc6ugOIoVwCgryB2KJNCnbBrlY6pq0W2Ww7sxSkUa9_WTPBIwAufhCQYkb7gA2eUbb6EIZjrl5mQ.sc3; ucid=hPzasmkDyTeHN0DinLRGvw; hasGmid=ver4; gig_bootstrap_3_L94eyQ-wvJhWm7Afp1oBhfTGXZArUfSHHW9p9Pncg513hZELXsxCfMWHrF8f5P5a=auth_ver4;glt_" + hello.ApiKey + "=" + login.SessionInfo.LoginToken,
	})

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if _, err := param(); err != nil {
		return nil, err
	}

	u, err = url.Parse(resp.Header.Get("location"))
	if err != nil {
		return nil, err
	}

	token := oauth2.Token{
		AccessToken:  u.Query().Get("access_token"),
		RefreshToken: u.Query().Get("refresh_token"),
		// Expiry: time.Now().Add(time.Duration(u.Query().Get("expires_in"))*time.Second),
	}

	ts := time.Now().Format(time.RFC3339)
	nonce := lo.RandomString(16, lo.AlphanumericCharset)

	params := url.Values{
		"needSharedCar": []string{"1"},
		"userId":        []string{login.SessionInfo.LoginToken},
	}
	uri = fmt.Sprintf("https://api.ecloudeu.com/device-platform/user/vehicle/secure?%s", params.Encode())
	sign, err := createSignature(nonce, params, ts, http.MethodGet, uri, "")
	if err != nil {
		return nil, err
	}

	deviceId := lo.RandomString(16, lo.AlphanumericCharset)
	req, _ = request.New(http.MethodGet, uri, nil, map[string]string{
		"x-app-id":                "SmartAPPEU",
		"accept":                  "application/json;responseformat=3",
		"x-agent-type":            "iOS",
		"x-device-type":           "mobile",
		"x-operator-code":         "SMART",
		"x-device-identifier":     deviceId,
		"x-env-type":              "production",
		"x-version":               "smartNew",
		"accept-language":         "en_US",
		"x-api-signature-version": "1.0",
		"x-api-signature-nonce":   nonce,
		"x-device-manufacture":    "Apple",
		"x-device-brand":          "Apple",
		"x-device-model":          "iPhone",
		"x-agent-version":         "17.1",
		"authorization":           token.AccessToken,
		"content-type":            "application/json; charset=utf-8",
		"user-agent":              "Hello smart/1.4.0 (iPhone; iOS 17.1; Scale/3.00)",
		"x-signature":             sign,
		"x-timestamp":             ts,
	})

	if err := client.DoJSON(req, nil); err != nil {
		return nil, err
	}

	_ = token
	return v, err
}

func createSignature(nonce string, params url.Values, ts, method, uri string, data any) (string, error) {
	md5 := md5.New()
	bytes := []byte("1B2M2Y8AsgTpgAmY7PhCfg==")
	if data != "" {
		var err error
		if bytes, err = json.Marshal(data); err != nil {
			return "", err
		}
	}

	if _, err := md5.Write(bytes); err != nil {
		return "", err
	}

	md5Hash := hex.EncodeToString(md5.Sum(nil))

	payload := fmt.Sprintf(`application/json;responseformat=3
x-api-signature-nonce:%s
x-api-signature-version:1.0

%s
%s
%s
%s
%s`, nonce, params.Encode(), md5Hash, ts, method, uri)

	secret, err := base64.StdEncoding.DecodeString("NzRlNzQ2OWFmZjUwNDJiYmJlZDdiYmIxYjM2YzE1ZTk=")
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha1.New, secret)
	if _, err := mac.Write([]byte(payload)); err != nil {
		return "", err
	}

	return hex.EncodeToString(mac.Sum(nil)), nil
}

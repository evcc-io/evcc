package hello

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

type Identity struct {
	*request.Helper
	oauth2.TokenSource
	user, password   string
	userID, deviceID string
}

func NewIdentity(log *util.Logger, user, password string) (*Identity, error) {
	v := &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
		deviceID: lo.RandomString(16, lo.AlphanumericCharset),
	}

	v.TokenSource = oauth.RefreshTokenSource(nil, v)

	_, err := v.Token()

	return v, err
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	token, err := v.login()
	if err != nil {
		return nil, err
	}

	appToken, userID, err := v.appToken(token)
	if err != nil {
		return nil, err
	}

	if err == nil {
		v.userID = userID
	}

	return appToken, err
}

func (v *Identity) DeviceID() string {
	return v.deviceID
}

func (v *Identity) UserID() (string, error) {
	var err error
	if v.userID == "" {
		err = errors.New("missing user id")
	}
	return v.userID, err
}

func (v *Identity) login() (*oauth2.Token, error) {
	uri := "https://awsapi.future.smart.com/login-app/api/v1/authorize?uiLocales=de-DE&uiLocales=de-DE"
	req, _ := request.New(http.MethodGet, uri, nil, map[string]string{
		"user-agent":       userAgent,
		"x-requested-with": "com.smart.hellosmart",
	})

	resp, err := v.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	u := resp.Request.URL

	data := url.Values{
		"loginID":           {v.user},
		"password":          {v.password},
		"sessionExpiration": {"2592000"},
		"targetEnv":         {"jssdk"},
		"include":           {"profile,data,emails,subscriptions,preferences"},
		"includeUserInfo":   {"true"},
		"loginMode":         {"standard"},
		"lang":              {"de"},
		"APIKey":            {ApiKey},
		"source":            {"showScreenSet"},
		"sdk":               {"js_latest"},
		"authMode":          {"cookie"},
		"pageURL":           {"https://app.id.smart.com/login?gig_ui_locales=de-DE"},
		"sdkBuild":          {"15482"},
		"format":            {"json"},
	}

	uri = "https://auth.smart.com/accounts.login"
	req, _ = request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), map[string]string{
		"user-agent":       userAgent,
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
		UserInfo struct {
			UID                           string
			FirstName, LastName, NickName string
		}
	}

	if err := v.DoJSON(req, &login); err != nil {
		return nil, err
	}
	if login.ErrorCode != 0 {
		return nil, fmt.Errorf("%s: %s", login.ErrorMessage, login.ErrorDetails)
	}
	defer resp.Body.Close()

	var param request.InterceptResult
	v.Client.CheckRedirect, param = request.InterceptRedirect("access_token", true)

	uri = fmt.Sprintf("https://auth.smart.com/oidc/op/v1.0/%s/authorize/continue?context=%s&login_token=%s", ApiKey, u.Query().Get("context"), login.SessionInfo.LoginToken)
	req, _ = request.New(http.MethodGet, uri, nil, map[string]string{
		"user-agent":       userAgent,
		"x-requested-with": "com.smart.hellosmart",
		"content-type":     request.FormContent,
		"cookie":           "gmid=gmid.ver4.AcbHPqUK5Q.xOaWPhRTb7gy-6-GUW6cxQVf_t7LhbmeabBNXqqqsT6dpLJLOWCGWZM07EkmfM4j.u2AMsCQ9ZsKc6ugOIoVwCgryB2KJNCnbBrlY6pq0W2Ww7sxSkUa9_WTPBIwAufhCQYkb7gA2eUbb6EIZjrl5mQ.sc3; ucid=hPzasmkDyTeHN0DinLRGvw; hasGmid=ver4; gig_bootstrap_3_L94eyQ-wvJhWm7Afp1oBhfTGXZArUfSHHW9p9Pncg513hZELXsxCfMWHrF8f5P5a=auth_ver4;glt_" + ApiKey + "=" + login.SessionInfo.LoginToken,
	})

	resp, err = v.Do(req)
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
	}

	return &token, nil
}

func (v *Identity) appToken(token *oauth2.Token) (*oauth2.Token, string, error) {
	params := url.Values{
		"identity_type": {"smart"},
	}

	data := map[string]string{
		"accessToken": token.AccessToken,
	}

	path := "/auth/account/session/secure"
	nonce, ts, sign, err := createSignature(http.MethodPost, path, params, request.MarshalJSON(data))
	if err != nil {
		return nil, "", err
	}

	uri := fmt.Sprintf("%s/%s?%s", ApiURI, strings.TrimPrefix(path, "/"), params.Encode())
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), map[string]string{
		"Accept":                  "application/json;responseformat=3",
		"Content-Type":            "application/json; charset=utf-8",
		"X-Api-Signature-Version": "1.0",
		"X-Api-Signature-Nonce":   nonce,
		"X-App-Id":                appID,
		"X-Device-Identifier":     v.deviceID,
		"X-Operator-Code":         operatorCode,
		"X-Signature":             sign,
		"X-Timestamp":             ts,
	})

	var res struct {
		Code    ResponseCode
		Message string
		Data    AppToken
	}

	if err := v.DoJSON(req, &res); err != nil {
		return nil, "", err
	} else if res.Code != ResponseOK {
		return nil, "", fmt.Errorf("%d: %s", res.Code, res.Message)
	}

	tok := oauth2.Token{
		AccessToken:  res.Data.AccessToken,
		RefreshToken: res.Data.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(res.Data.ExpiresIn) * time.Second),
	}

	return &tok, res.Data.UserId, nil
}

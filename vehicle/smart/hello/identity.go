package hello

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

// savedState holds the identity data persisted across evcc restarts.
type savedState struct {
	Token    oauth2.Token `json:"token"`
	UserID   string       `json:"userId"`
	DeviceID string       `json:"deviceId"`
}

type Identity struct {
	*request.Helper
	oauth2.TokenSource
	log              *util.Logger
	user, password   string
	userID, deviceID string
	subject          string
	mu               sync.Mutex
}

func NewIdentity(log *util.Logger, user, password string) (*Identity, error) {
	v := &Identity{
		Helper:   request.NewHelper(log),
		log:      log,
		user:     user,
		password: password,
		subject:  "smart-hello." + user,
	}

	var state savedState
	if err := settings.Json(v.subject, &state); err != nil {
		if !errors.Is(err, settings.ErrNotFound) {
			v.log.WARN.Printf("load state: %v", err)
		}
		// no usable persisted state — generate a fresh device ID (sent in login headers)
		state.DeviceID = lo.RandomString(16, lo.AlphanumericCharset)
	}

	// deviceID must be set before any login — it is sent in request headers.
	v.deviceID = state.DeviceID
	v.userID = state.UserID

	var token *oauth2.Token
	if state.Token.Valid() {
		token = &state.Token
	} else {
		var err error
		token, err = v.refreshToken(nil)
		if err != nil {
			return nil, err
		}
	}

	v.TokenSource = oauth.RefreshTokenSource(token, v.refreshToken)
	return v, nil
}

func (v *Identity) refreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	tok, err := v.login()
	if err != nil {
		return nil, err
	}

	appToken, userID, err := v.appToken(tok)
	if err != nil {
		return nil, err
	}

	v.userID = userID

	if err := settings.SetJson(v.subject, savedState{
		Token:    *appToken,
		UserID:   userID,
		DeviceID: v.deviceID,
	}); err != nil {
		return nil, err
	}

	return appToken, nil
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
	if err == nil && resp.StatusCode != 200 {
		err = fmt.Errorf("status: %d", resp.StatusCode)
	}
	if err != nil {
		return nil, fmt.Errorf("authorize: %w", err)
	}
	defer resp.Body.Close()

	context := resp.Request.URL.Query().Get("context")
	if context == "" {
		return nil, fmt.Errorf("missing context: %s", resp.Request.URL.String())
	}

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
		return nil, fmt.Errorf("accounts.login: %w", err)
	}
	if login.ErrorCode != 0 {
		return nil, fmt.Errorf("%s: %s", login.ErrorMessage, login.ErrorDetails)
	}
	defer resp.Body.Close()

	var param request.InterceptResult
	v.Client.CheckRedirect, param = request.InterceptRedirect("access_token", true)

	uri = fmt.Sprintf("https://auth.smart.com/oidc/op/v1.0/%s/authorize/continue?context=%s&login_token=%s", ApiKey, context, login.SessionInfo.LoginToken)
	req, _ = request.New(http.MethodGet, uri, nil, map[string]string{
		"user-agent":       userAgent,
		"x-requested-with": "com.smart.hellosmart",
		"content-type":     request.FormContent,
		"cookie":           "gmid=gmid.ver4.AcbHPqUK5Q.xOaWPhRTb7gy-6-GUW6cxQVf_t7LhbmeabBNXqqqsT6dpLJLOWCGWZM07EkmfM4j.u2AMsCQ9ZsKc6ugOIoVwCgryB2KJNCnbBrlY6pq0W2Ww7sxSkUa9_WTPBIwAufhCQYkb7gA2eUbb6EIZjrl5mQ.sc3; ucid=hPzasmkDyTeHN0DinLRGvw; hasGmid=ver4; gig_bootstrap_3_L94eyQ-wvJhWm7Afp1oBhfTGXZArUfSHHW9p9Pncg513hZELXsxCfMWHrF8f5P5a=auth_ver4;glt_" + ApiKey + "=" + login.SessionInfo.LoginToken,
	})

	resp, err = v.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	if _, err := param(); err != nil {
		return nil, err
	}

	u, err := url.Parse(resp.Header.Get("location"))
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
		Code    Int
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

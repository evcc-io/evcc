// Package chargepoint implements authentication for the ChargePoint Home Flex
// charger.
//
// # Authentication overview
//
// Behavior here is modeled after version 6.20.1 of the iOS app and
// sso.chargepoint.com.
//
// ChargePoint uses username/password credentials and expects two tokens.
// Interacting with the charger involves different subdomains which have
// slightly different expectations around these tokens.
//
// The accounts API endpoint (<account>/v2/driver/profile/account/login) is
// used with a stable iOS device fingerprint (derived from username). On
// success, the endpoint returns a "sessionId" that encodes the region directly
// in its structure: "<token>#D<?????>#R<region>". It also returns a
// "ssoSessionId" JWT token, valid for 6 months. This seems to match the
// behavior on sso.chargepoint.com's login endpoint.
//
// # CAPTCHA protection
//
// All ChargePoint endpoints are protected by DataDome bot detection. Repeated
// logins from the same IP (~4 per half hour) will trigger a CAPTCHA challenge
// and return HTTP 403 even for valid credentials. Only after a cooldown of
// several hours will you be able to login again, however previously generated
// credentials will continue to work. It's for this reason that we always
// prefer persisted DB credentials over new logins.

package chargepoint

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/google/uuid"
	"golang.org/x/net/publicsuffix"
)

const (
	discoveryAPI = "https://discovery.chargepoint.com/discovery/v3/globalconfig"
	appVersion   = "6.20.1"
	userAgent    = "com.coulomb.ChargePoint/" + appVersion + " CFNetwork/3860.400.51 Darwin/25.3.0"
)

// Identity manages ChargePoint session state using a shared cookie jar,
// mirroring how python-chargepoint uses requests.Session.
type Identity struct {
	*request.Helper
	identityState

	settingsKey string
	deviceData  DeviceData
	cfg         *globalConfig
}

// identityState is persisted in settings.
type identityState struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	UserID       int32  `json:"user_id"`
	Region       string `json:"region"`
	SessionID    string `json:"sessionId"`
	SSOSessionID string `json:"ssoSessionId"` // JWT returned by login; sso.chargepoint.com also returns this token.
}

// NewIdentity creates a ChargePoint Identity backed by a cookie jar. It loads
// a stored session from settings if available, then refreshes it; otherwise
// falls back to a fresh login.
func NewIdentity(log *util.Logger, username, password string) (*Identity, error) {
	v := &Identity{
		Helper:      request.NewHelper(log),
		settingsKey: "chargepoint." + username,
		deviceData:  newDeviceData(username),

		identityState: identityState{
			Username: username,
			Password: password,
		},
	}

	v.Helper.Jar, _ = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	cfg, err := discover(v.Helper, v.deviceData, v.Username)
	if err != nil {
		return nil, fmt.Errorf("discovering endpoints: %w", err)
	}
	v.cfg = cfg

	if err := v.Login(); err != nil {
		return nil, err
	}

	return v, nil
}

// Login performs the ChargePoint mobile app login flow.
func (v *Identity) Login() error {
	var state identityState
	if err := settings.Json(v.settingsKey, &state); err == nil &&
		state.SSOSessionID != "" && !jwtExpired(state.SSOSessionID) {
		v.UserID = state.UserID
		v.Region = state.Region
		v.SessionID = state.SessionID
		v.SSOSessionID = state.SSOSessionID
		if err := v.validate(); err == nil {
			return nil
		}
	}

	data := struct {
		DeviceData DeviceData `json:"deviceData"`
		Username   string     `json:"username"`
		Password   string     `json:"password"`
	}{v.deviceData, v.Username, v.Password}

	uri := v.cfg.EndPoints.Accounts.Value + "v2/driver/profile/account/login"
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("User-Agent", userAgent)

	var res accountLoginResponse
	if err := v.Helper.DoJSON(req, &res); err != nil {
		return fmt.Errorf("logging in: %w", err)
	}

	if res.SessionID == "" {
		return fmt.Errorf("no session ID in login response")
	}

	v.UserID = res.User.UserID
	v.Region = v.cfg.Region
	v.SessionID = res.SessionID
	v.SSOSessionID = res.SSOSessionID

	if err := settings.SetJson(v.settingsKey, v.identityState); err != nil {
		return fmt.Errorf("persisting chargepoint identity: %w", err)
	}

	return nil
}

// validate checks whether the current credentials are still valid by fetching
// the user profile. Returns nil on success.
func (v *Identity) validate() error {
	headers := map[string]string{
		"Accept-Encoding":  "gzip, deflate",
		"User-Agent":       userAgent,
		"CP-Region":        v.Region,
		"CP-Session-Token": v.SessionID,
		"CP-Session-Type":  "CP_SESSION_TOKEN",
		"Cache-Control":    "no-store",
		"Accept-Language":  "en;q=1",
		"Cookie":           "coulomb_sess=" + v.SessionID + "; auth-session=" + v.SSOSessionID,
	}
	req, err := request.New(http.MethodGet, v.cfg.EndPoints.Accounts.Value+"v1/driver/profile/user", nil,
		request.JSONEncoding, headers)
	if err != nil {
		return err
	}
	return v.Helper.DoJSON(req, nil)
}

// jwtExpired returns true if the JWT's exp claim is in the past or the token
// cannot be parsed. The signature is not verified — we only need the expiry.
func jwtExpired(tokenStr string) bool {
	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		return true
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return true
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return true
	}
	return time.Now().Unix() > claims.Exp
}

func discover(c *request.Helper, dev DeviceData, username string) (*globalConfig, error) {
	data := struct {
		DeviceData DeviceData `json:"deviceData"`
		Username   string     `json:"username"`
	}{dev, username}

	req, err := request.New(http.MethodPost, discoveryAPI, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return nil, err
	}

	var cfg globalConfig
	if err := c.DoJSON(req, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// deviceUDID returns a stable UUID v5 derived from the username,
// mimicking a real iOS device that always presents the same UDID.
func deviceUDID(username string) string {
	return uuid.NewSHA1(uuid.NameSpaceX500, []byte(username)).String()
}

// newDeviceData returns a stable iOS device fingerprint derived from the username.
func newDeviceData(username string) DeviceData {
	return DeviceData{
		AppID:              "com.coulomb.ChargePoint",
		Manufacturer:       "Apple",
		Model:              "iPhone",
		NotificationID:     "",
		NotificationIDType: "",
		Type:               "IOS",
		UDID:               deviceUDID(username),
		Version:            appVersion,
	}
}

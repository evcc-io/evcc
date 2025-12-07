package bluelink

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

const (
	DeviceIdURL        = "/api/v1/spa/notifications/register"
	IntegrationInfoURL = "/api/v1/user/integrationinfo"
	SilentSigninURL    = "/api/v1/user/silentsignin"
	LanguageURL        = "/api/v1/user/language"
	LoginURL           = "/api/v1/user/signin"
	TokenURL           = "/auth/api/v2/user/oauth2/token"
)

// Config is the bluelink API configuration
type Config struct {
	URI               string
	BasicToken        string
	CCSPServiceID     string
	CCSPApplicationID string
	CCSPServiceSecret string
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
	data := map[string]any{
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

// RefreshToken implements oauth.TokenRefresher
func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	var res oauth2.Token

	uri := v.config.LoginFormHost + TokenURL
	headers := map[string]string{
		"Content-type": "application/x-www-form-urlencoded",
		"User-Agent":   "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.166 Mobile Safari/535.19_CCS_APP_AOS",
	}
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
		"client_id":     {v.config.CCSPServiceID},
		"client_secret": {v.config.CCSPServiceSecret},
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

	switch brand {
	case "kia":
	case "hyundai":
	default:
		return fmt.Errorf("unknown brand (%s)", brand)
	}

	token, err := v.RefreshToken(&oauth2.Token{RefreshToken: password})
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	v.TokenSource = oauth.RefreshTokenSource(token, v)

	v.deviceID, err = v.getDeviceID()
	if err != nil {
		return fmt.Errorf("error getting device id: %w", err)
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

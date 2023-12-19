package hello

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/samber/lo"
)

// https://github.com/TA2k/ioBroker.smart-eq

type API struct {
	*request.Helper
	identity *Identity
	deviceId string
}

func NewAPI(log *util.Logger, identity *Identity) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		deviceId: lo.RandomString(16, lo.AlphanumericCharset),
		identity: identity,
	}

	// replace client transport with authenticated transport
	v.Client.Transport = &transport.Decorator{
		Base: v.Client.Transport,

		// Decorator: transport.DecorateHeaders(map[string]string{
		// }),

		Decorator: func(req *http.Request) error {
			token, err := identity.Token()
			if err != nil {
				return err
			}

			req.Header.Set("authorization", token.AccessToken)
			return nil
		},
	}

	return v
}

func (v *API) Vehicles() ([]string, error) {
	var res struct {
		Code    ResponseCode
		Message string
		Data    struct {
			List []Vehicle
		}
	}

	path := "/device-platform/user/vehicle/secure"

	userID, err := v.identity.UserID()
	if err != nil {
		return nil, err
	}

	params := url.Values{
		"needSharedCar": []string{"1"},
		"userId":        []string{userID},
	}

	uri, nonce, ts, sign, err := createSignature(params, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	req, _ := request.New(http.MethodGet, uri, nil, map[string]string{
		"x-app-id":                "SmartAPPEU",
		"accept":                  "application/json;responseformat=3",
		"x-agent-type":            "iOS",
		"x-device-type":           "mobile",
		"x-operator-code":         "SMART",
		"x-device-identifier":     v.deviceId,
		"x-env-type":              "production",
		"x-version":               "smartNew",
		"accept-language":         "en_US",
		"x-api-signature-version": "1.0",
		"x-api-signature-nonce":   nonce,
		"x-device-manufacture":    "Apple",
		"x-device-brand":          "Apple",
		"x-device-model":          "iPhone",
		"x-agent-version":         "17.1",
		"content-type":            "application/json; charset=utf-8",
		"user-agent":              "Hello smart/1.4.0 (iPhone; iOS 17.1; Scale/3.00)",
		"x-signature":             sign,
		"x-timestamp":             ts,
	})

	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	} else if res.Code != ResponseOK {
		return nil, fmt.Errorf("%d: %s", res.Code, res.Message)
	}

	vehicles := lo.Map(res.Data.List, func(v Vehicle, _ int) string {
		return v.VIN
	})

	return vehicles, err
}

func createSignature(params url.Values, method, path string, post any) (string, string, string, string, error) {
	nonce := lo.RandomString(16, lo.AlphanumericCharset)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)

	md5Hash := "1B2M2Y8AsgTpgAmY7PhCfg=="
	if post != nil {
		bytes, err := json.Marshal(post)
		if err != nil {
			return "", "", "", "", err
		}

		hash := md5.New()
		hash.Write(bytes)
		md5Hash = base64.StdEncoding.EncodeToString(hash.Sum(nil))
	}

	payload := fmt.Sprintf(`application/json;responseformat=3
x-api-signature-nonce:%s
x-api-signature-version:1.0

%s
%s
%s
%s
%s`, nonce, params.Encode(), md5Hash, ts, method, path)

	secret, err := base64.StdEncoding.DecodeString("NzRlNzQ2OWFmZjUwNDJiYmJlZDdiYmIxYjM2YzE1ZTk=")
	if err != nil {
		return "", "", "", "", err
	}

	mac := hmac.New(sha1.New, secret)
	mac.Write([]byte(payload))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	uri := fmt.Sprintf("%s/%s?%s", ApiURI, strings.TrimPrefix(path, "/"), params.Encode())

	return uri, nonce, ts, sign, nil
}

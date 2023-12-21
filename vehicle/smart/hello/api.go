package hello

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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

	v.Client.Transport = &transport.Decorator{
		Base: &transport.Decorator{
			Base: v.Client.Transport,

			// decorate token
			Decorator: func(req *http.Request) error {
				token, err := identity.Token()
				if err != nil {
					return err
				}

				req.Header.Set("authorization", token.AccessToken)
				return nil
			},
		},

		// decorate headers
		Decorator: transport.DecorateHeaders(map[string]string{
			"x-app-id":                "SmartAPPEU",
			"accept":                  "application/json;responseformat=3",
			"x-agent-type":            "iOS",
			"x-device-type":           "mobile",
			"x-operator-code":         "SMART",
			"x-env-type":              "production",
			"x-version":               "smartNew",
			"accept-language":         "en_US",
			"x-api-signature-version": "1.0",
			"x-device-manufacture":    "Apple",
			"x-device-brand":          "Apple",
			"x-device-identifier":     v.deviceId,
			"x-device-model":          "iPhone",
			"x-agent-version":         "17.1",
			"content-type":            "application/json; charset=utf-8",
			"user-agent":              "Hello smart/1.4.0 (iPhone; iOS 17.1; Scale/3.00)",
		}),
	}

	return v
}

func (v *API) request(method, path string, params url.Values, body io.Reader) (*http.Request, error) {
	if body != nil {
		b, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}
		// read from buffer
		body = bytes.NewReader(b)
	}

	nonce, ts, sign, err := createSignature(method, path, params, body)
	if err != nil {
		return nil, err
	}

	if body != nil {
		// rewind body
		body.(*bytes.Reader).Seek(0, io.SeekStart)
	}

	uri := fmt.Sprintf("%s/%s?%s", ApiURI, strings.TrimPrefix(path, "/"), params.Encode())
	req, err := request.New(method, uri, body, map[string]string{
		"x-api-signature-nonce": nonce,
		"x-signature":           sign,
		"x-timestamp":           ts,
	})

	return req, err
}

func (v *API) Vehicles() ([]string, error) {
	var res struct {
		Code    ResponseCode
		Message string
		Data    struct {
			List []Vehicle
		}
	}

	userID, err := v.identity.UserID()
	if err != nil {
		return nil, err
	}

	params := url.Values{
		"needSharedCar": []string{"1"},
		"userId":        []string{userID},
	}

	path := "/device-platform/user/vehicle/secure"
	req, err := v.request(http.MethodGet, path, params, nil)
	if err != nil {
		return nil, err
	}

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

func (v *API) Status(vin string) (VehicleStatus, error) {
	var res struct {
		Code    ResponseCode
		Message string
		Data    struct {
			VehicleStatus VehicleStatus
		}
	}

	userID, err := v.identity.UserID()
	if err != nil {
		return VehicleStatus{}, err
	}

	params := url.Values{
		"latest": []string{"true"},
		"target": []string{""},
		"userId": []string{userID},
	}

	path := "/remote-control/vehicle/status/" + vin
	req, err := v.request(http.MethodGet, path, params, nil)
	if err != nil {
		return VehicleStatus{}, err
	}

	if err := v.DoJSON(req, &res); err != nil {
		return VehicleStatus{}, err
	} else if res.Code != ResponseOK {
		return VehicleStatus{}, fmt.Errorf("%d: %s", res.Code, res.Message)
	}

	return res.Data.VehicleStatus, err
}

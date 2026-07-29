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
)

// https://github.com/TA2k/ioBroker.smart-eq

type API struct {
	*request.Helper
	identity *Identity
	baseURI  string
}

func NewAPI(log *util.Logger, identity *Identity) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		identity: identity,
		baseURI:  ApiURI,
	}

	v.Client.Transport = &transport.Decorator{
		Base: &transport.Decorator{
			Base: v.Client.Transport,

			// decorate token
			Decorator: func(req *http.Request) error {
				token, err := identity.Token()
				if err == nil {
					req.Header.Set("authorization", token.AccessToken)
				}
				return err
			},
		},

		// decorate headers
		Decorator: transport.DecorateHeaders(map[string]string{
			"accept":                  "application/json;responseformat=3",
			"content-type":            "application/json; charset=utf-8",
			"x-operator-code":         operatorCode,
			"x-api-signature-version": "1.0",
			"x-app-id":                appID,
			"x-device-identifier":     v.identity.DeviceID(),
		}),
	}

	return v
}

// SetSeries selects the API host for the vehicle's platform.
// Smart #5 (series HY) is served by the V2 host; #1/#3 (HX/HC) use V1.
func (v *API) SetSeries(series string) {
	if strings.HasPrefix(series, "HY") {
		v.baseURI = ApiURIV2
	}
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

	uri := fmt.Sprintf("%s/%s?%s", v.baseURI, strings.TrimPrefix(path, "/"), params.Encode())
	req, err := request.New(method, uri, body, map[string]string{
		"x-api-signature-nonce": nonce,
		"x-signature":           sign,
		"x-timestamp":           ts,
	})

	return req, err
}

func (v *API) Vehicles() ([]Vehicle, error) {
	var res struct {
		Code    Int
		Message string
		Error   Error
		Data    struct {
			List []Vehicle
		}
	}

	userID, err := v.identity.UserID()
	if err != nil {
		return nil, err
	}

	params := url.Values{
		"needSharedCar": {"1"},
		"userId":        {userID},
	}

	// vehicle list is fetched on V1: SetSeries runs only after this call
	path := "/device-platform/user/vehicle/secure"
	req, err := v.request(http.MethodGet, path, params, nil)
	if err != nil {
		return nil, err
	}

	err = v.DoJSON(req, &res)
	if err := responseError(err, res.Code, res.Message, res.Error); err != nil {
		return nil, err
	}

	return res.Data.List, err
}

func (v *API) UpdateSession(vin string) error {
	token, err := v.identity.Token()
	if err != nil {
		return err
	}

	params := url.Values{
		"identity_type": {"smart"},
	}

	data := map[string]string{
		"vin":          vin,
		"sessionToken": token.AccessToken,
		"language":     "",
	}

	path := "/device-platform/user/session/update"
	req, err := v.request(http.MethodPost, path, params, request.MarshalJSON(data))
	if err != nil {
		return err
	}

	var res struct {
		Code    Int
		Message string
		Error   Error
	}

	err = v.DoJSON(req, &res)
	return responseError(err, res.Code, res.Message, res.Error)
}

func (v *API) Status(vin string) (VehicleStatus, error) {
	if err := v.UpdateSession(vin); err != nil {
		return VehicleStatus{}, fmt.Errorf("update session failed: %w", err)
	}

	var res struct {
		Code    Int
		Message string
		Error   Error
		Data    struct {
			VehicleStatus VehicleStatus
		}
	}

	userID, err := v.identity.UserID()
	if err != nil {
		return VehicleStatus{}, err
	}

	params := url.Values{
		"latest": {"true"},
		"target": {""},
		"userId": {userID},
	}

	path := "/remote-control/vehicle/status/" + vin
	req, err := v.request(http.MethodGet, path, params, nil)
	if err != nil {
		return VehicleStatus{}, err
	}

	err = v.DoJSON(req, &res)
	if err := responseError(err, res.Code, res.Message, res.Error); err != nil {
		return VehicleStatus{}, err
	}

	return res.Data.VehicleStatus, err
}

func (v *API) SocStatus(vin string) (VehicleSocStatus, error) {
	if err := v.UpdateSession(vin); err != nil {
		return VehicleSocStatus{}, fmt.Errorf("update session failed: %w", err)
	}

	var res struct {
		Code    Int
		Message string
		Error   Error
		Data    VehicleSocStatus
	}

	params := url.Values{
		"setting": {"charging"},
	}

	path := "/remote-control/vehicle/status/soc/" + vin
	req, err := v.request(http.MethodGet, path, params, nil)
	if err != nil {
		return VehicleSocStatus{}, err
	}

	err = v.DoJSON(req, &res)
	if err := responseError(err, res.Code, res.Message, res.Error); err != nil {
		return VehicleSocStatus{}, err
	}

	return res.Data, err
}

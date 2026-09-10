package hello

import (
	"bytes"
	"encoding/json"
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

// response is the common envelope of all api responses
type response[T any] struct {
	Code    Int
	Message string
	Error   Error
	Data    T
}

// do executes a signed request. A ResponseTokenInvalid answer means the backend
// no longer accepts the token although it has not expired yet- retry once with a
// fresh login. The body is built per attempt as it may embed the current token.
func do[T any](v *API, method, path string, params url.Values, body func() ([]byte, error)) (T, error) {
	var zero T
	var res response[T]
	var err error

	for range 2 {
		var rdr io.Reader
		if body != nil {
			b, err := body()
			if err != nil {
				return zero, err
			}
			rdr = bytes.NewReader(b)
		}

		var req *http.Request
		req, err = v.request(method, path, params, rdr)
		if err != nil {
			return zero, err
		}

		res = response[T]{}
		err = v.DoJSON(req, &res)

		if res.Code != ResponseTokenInvalid && res.Error.Code != ResponseTokenInvalid {
			break
		}

		v.identity.Invalidate()
	}

	if err := responseError(err, res.Code, res.Message, res.Error); err != nil {
		return zero, err
	}

	return res.Data, nil
}

func (v *API) Vehicles() ([]Vehicle, error) {
	userID, err := v.identity.UserID()
	if err != nil {
		return nil, err
	}

	params := url.Values{
		"needSharedCar": {"1"},
		"userId":        {userID},
	}

	// vehicle list is fetched on V1: SetSeries runs only after this call
	res, err := do[struct{ List []Vehicle }](v, http.MethodGet, "/device-platform/user/vehicle/secure", params, nil)

	return res.List, err
}

func (v *API) UpdateSession(vin string) error {
	params := url.Values{
		"identity_type": {"smart"},
	}

	body := func() ([]byte, error) {
		token, err := v.identity.Token()
		if err != nil {
			return nil, err
		}

		return json.Marshal(map[string]string{
			"vin":          vin,
			"sessionToken": token.AccessToken,
			"language":     "",
		})
	}

	_, err := do[struct{}](v, http.MethodPost, "/device-platform/user/session/update", params, body)

	return err
}

func (v *API) Status(vin string) (VehicleStatus, error) {
	if err := v.UpdateSession(vin); err != nil {
		return VehicleStatus{}, fmt.Errorf("update session failed: %w", err)
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

	res, err := do[struct{ VehicleStatus VehicleStatus }](v, http.MethodGet, "/remote-control/vehicle/status/"+vin, params, nil)

	return res.VehicleStatus, err
}

func (v *API) SocStatus(vin string) (VehicleSocStatus, error) {
	if err := v.UpdateSession(vin); err != nil {
		return VehicleSocStatus{}, fmt.Errorf("update session failed: %w", err)
	}

	params := url.Values{
		"setting": {"charging"},
	}

	return do[VehicleSocStatus](v, http.MethodGet, "/remote-control/vehicle/status/soc/"+vin, params, nil)
}

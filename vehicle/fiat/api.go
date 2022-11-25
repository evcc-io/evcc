package fiat

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
)

const (
	ApiURI  = "https://channels.sdpr-01.fcagcv.com"
	ApiKey  = "3_mOx_J2dRgjXYCdyhchv3b5lhi54eBcdCTX4BI8MORqmZCoQWhA0mV2PTlptLGUQI"
	XApiKey = "qLYupk65UU1tw2Ih1cJhs4izijgRDbir2UFHA3Je"

	AuthURI     = "https://mfa.fcl-01.fcagcv.com"
	XAuthApiKey = "JWRYW7IYhW9v0RqDghQSx4UcRYRILNmc8zAuh5ys"
)

// API is an api.Vehicle implementation for Fiat cars
type API struct {
	identity *Identity
	*request.Helper
}

func NewAPI(log *util.Logger, identity *Identity) *API {
	api := &API{
		identity: identity,
		Helper:   request.NewHelper(log),
	}

	return api
}

func (v *API) request(method, uri string, body io.ReadSeeker) (*http.Request, error) {
	headers := map[string]string{
		"Content-Type":        "application/json",
		"Accept":              "application/json",
		"X-Clientapp-Version": "1.0",
		"ClientrequestId":     lo.RandomString(16, lo.LettersCharset),
		"X-Api-Key":           XApiKey,
		"X-Originator-Type":   "web",
		"locale":              "de_de", // only required for pinAuth
	}

	req, err := request.New(method, uri, body, headers)
	if err == nil {
		// hack for pinAuth method
		if strings.HasPrefix(uri, AuthURI) {
			req.Header.Set("X-Api-Key", XAuthApiKey)
		}

		err = v.identity.Sign(req, body)
	}

	return req, err
}

func (v *API) Vehicles() ([]string, error) {
	var res VehiclesResponse

	uri := fmt.Sprintf("%s/v4/accounts/%s/vehicles?stage=ALL", ApiURI, v.identity.UID())

	req, err := v.request(http.MethodGet, uri, nil)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	var vehicles []string
	if err == nil {
		for _, v := range res.Vehicles {
			vehicles = append(vehicles, v.VIN)
		}
	}

	return vehicles, err
}

func (v *API) Status(vin string) (StatusResponse, error) {
	var res StatusResponse

	uri := fmt.Sprintf("%s/v2/accounts/%s/vehicles/%s/status", ApiURI, v.identity.UID(), vin)

	req, err := v.request(http.MethodGet, uri, nil)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

func (v *API) Location(vin string) (LocationResponse, error) {
	var res LocationResponse

	uri := fmt.Sprintf("%s/v1/accounts/%s/vehicles/%s/location/lastknown", ApiURI, v.identity.UID(), vin)

	req, err := v.request(http.MethodGet, uri, nil)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

func (v *API) pinAuth(pin string) (string, error) {
	var res PinAuthResponse

	uri := fmt.Sprintf("%s/v1/accounts/%s/ignite/pin/authenticate", AuthURI, v.identity.UID())

	data := struct {
		PIN string `json:"pin"`
	}{
		PIN: base64.StdEncoding.EncodeToString([]byte(pin)),
	}

	req, err := v.request(http.MethodPost, uri, request.MarshalJSON(data))
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	if err == nil && res.Message != "" {
		err = fmt.Errorf("pin auth: %s", res.Message)
	}

	return res.Token, err
}

func (v *API) Action(vin, pin, action, cmd string) (ActionResponse, error) {
	var res ActionResponse

	token, err := v.pinAuth(pin)
	if err != nil {
		return res, err
	}

	uri := fmt.Sprintf("%s/v1/accounts/%s/vehicles/%s/%s", ApiURI, v.identity.UID(), vin, action)

	data := struct {
		Command string `json:"command"`
		PinAuth string `json:"pinAuth"`
	}{
		Command: cmd,
		PinAuth: token,
	}

	req, err := v.request(http.MethodPost, uri, request.MarshalJSON(data))
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	if err == nil && res.Message != "" {
		err = fmt.Errorf("action %s: %s", action, res.Message)
	}

	return res, err
}

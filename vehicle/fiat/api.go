package fiat

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const (
	ApiURI  = "https://channels.sdpr-01.fcagcv.com"
	ApiKey  = "3_mOx_J2dRgjXYCdyhchv3b5lhi54eBcdCTX4BI8MORqmZCoQWhA0mV2PTlptLGUQI"
	XApiKey = "qLYupk65UU1tw2Ih1cJhs4izijgRDbir2UFHA3Je"
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
		"ClientrequestId":     util.RandomString(16),
		"X-Api-Key":           XApiKey,
		"X-Originator-Type":   "web",
	}

	req, err := request.New(method, uri, body, headers)
	if err == nil {
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

func (v *API) pinAuth(pin string) (string, error) {
	var res PinResponse

	uri := fmt.Sprintf("%s/v1/accounts/%s/ignite/pin/authenticate", ApiURI, v.identity.UID())

	data := struct {
		PIN string `json:"pin"`
	}{
		PIN: base64.StdEncoding.EncodeToString([]byte(pin)),
	}

	b, err := json.Marshal(data)

	var req *http.Request
	if err == nil {
		req, err = v.request(http.MethodPost, uri, bytes.NewReader(b))
	}

	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res.Data.Token, err
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

	b, err := json.Marshal(data)

	var req *http.Request
	if err == nil {
		req, err = v.request(http.MethodPost, uri, bytes.NewReader(b))
	}

	if err == nil {
		err = v.DoJSON(req, &res)
	}

	if err == nil && res.Message != "" {
		err = fmt.Errorf("action %s failed: %s", action, res.Message)
	}

	return res, err
}

package kamereon

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/renault/gigya"
	"github.com/evcc-io/evcc/vehicle/renault/keys"
)

const (
	ActionStart = "start"
	ActionStop  = "stop"
)

type API struct {
	*request.Helper
	keys     keys.ConfigServer
	identity *gigya.Identity
	login    func() error
}

func New(log *util.Logger, keys keys.ConfigServer, identity *gigya.Identity, login func() error) *API {
	return &API{
		Helper:   request.NewHelper(log),
		keys:     keys,
		identity: identity,
		login:    login,
	}
}

func (v *API) request_(uri string, body io.Reader) (Response, error) {
	params := url.Values{"country": []string{"DE"}}
	headers := map[string]string{
		"x-gigya-id_token": v.identity.Token,
		"apikey":           v.keys.APIKey,
	}

	method := http.MethodGet
	if body != nil {
		method = http.MethodPost
	}

	var res Response
	req, err := request.New(method, uri+"?"+params.Encode(), body, headers)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

func (v *API) request(uri string, body io.Reader) (Response, error) {
	res, err := v.request_(uri, body)
	// repeat auth if error
	if err != nil {
		if err = v.login(); err == nil {
			res, err = v.request_(uri, nil)
		}
	}

	return res, err
}

func (v *API) Person(personID, brand string) (string, error) {
	uri := fmt.Sprintf("%s/commerce/v1/persons/%s", v.keys.Target, personID)
	res, err := v.request(uri, nil)
	if err != nil {
		return "", err
	}

	for _, account := range res.Accounts {
		if strings.Contains(strings.ToLower(account.AccountType), strings.ToLower(brand)) {
			return account.AccountID, nil
		}
	}

	return "", errors.New("account not found")
}

func (v *API) Vehicles(accountID string) ([]Vehicle, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/vehicles", v.keys.Target, accountID)
	res, err := v.request(uri, nil)
	return res.VehicleLinks, err
}

// Battery provides battery-status api response
func (v *API) Battery(accountID string, vin string) (Response, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kca/car-adapter/v2/cars/%s/battery-status", v.keys.Target, accountID, vin)
	return v.request(uri, nil)
}

// Hvac provides hvac-status api response
func (v *API) Hvac(accountID string, vin string) (Response, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kca/car-adapter/v1/cars/%s/hvac-status", v.keys.Target, accountID, vin)
	return v.request(uri, nil)
}

// Cockpit provides cockpit api response
func (v *API) Cockpit(accountID string, vin string) (Response, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kca/car-adapter/v2/cars/%s/cockpit", v.keys.Target, accountID, vin)
	return v.request(uri, nil)
}

func (v *API) WakeUp(accountID string, vin string) (Response, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kcm/v1/vehicles/%s/charge/pause-resume", v.keys.Target, accountID, vin)

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "ChargePauseResume",
			"attributes": map[string]interface{}{
				"action": "resume",
			},
		},
	}

	return v.request(uri, request.MarshalJSON(data))
}

func (v *API) Position(accountID string, vin string) (Response, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kca/car-adapter/v1/cars/%s/location", v.keys.Target, accountID, vin)
	return v.request(uri, nil)
}

func (v *API) Action(accountID, action string, vin string) (Response, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kca/car-adapter/v1/cars/%s/actions/charging-start", v.keys.Target, accountID, vin)

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "ChargingStart",
			"attributes": map[string]interface{}{
				"action": action,
			},
		},
	}

	return v.request(uri, request.MarshalJSON(data))
}

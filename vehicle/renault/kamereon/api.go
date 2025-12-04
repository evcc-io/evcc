package kamereon

import (
	"errors"
	"fmt"
	"net/http"
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

func NewAPI(log *util.Logger, keys keys.ConfigServer, identity *gigya.Identity, login func() error) *API {
	v := &API{
		Helper:   request.NewHelper(log),
		keys:     keys,
		identity: identity,
		login:    login,
	}

	v.Client.Transport = &AuthDecorator{
		Login:    v.login,
		Keys:     v.keys,
		Identity: v.identity,
		Base:     v.Client.Transport,
	}

	return v
}

func (v *API) Accounts(personID string) ([]Account, error) {
	uri := fmt.Sprintf("%s/commerce/v1/persons/%s", v.keys.Target, personID)

	var res struct {
		Accounts []Account `json:"accounts"`
	}
	err := v.GetJSON(uri, &res)

	return res.Accounts, err
}

func (v *API) AccountID(personID, brand string) (string, error) {
	accounts, err := v.Accounts(personID)

	if err != nil {
		return "", err
	}

	for _, account := range accounts {
		if strings.Contains(strings.ToLower(account.AccountType), strings.ToLower(brand)) {
			return account.AccountID, nil
		}
	}

	return "", errors.New("account not found")
}

func (v *API) Vehicles(accountID string) ([]Vehicle, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/vehicles", v.keys.Target, accountID)

	var res struct {
		VehicleLinks []Vehicle `json:"vehicleLinks"`
	}
	err := v.GetJSON(uri, &res)

	return res.VehicleLinks, err
}

func (v *API) BatteryStatus(accountID string, vin string) (BatteryStatus, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kca/car-adapter/v2/cars/%s/battery-status", v.keys.Target, accountID, vin)

	var res struct {
		Data struct {
			Attributes BatteryStatus `json:"attributes"`
		} `json:"data"`
	}
	err := v.GetJSON(uri, &res)

	return res.Data.Attributes, err
}

func (v *API) HvacStatus(accountID string, vin string) (HvacStatus, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kca/car-adapter/v1/cars/%s/hvac-status", v.keys.Target, accountID, vin)

	var res struct {
		Data struct {
			Attributes HvacStatus `json:"attributes"`
		} `json:"data"`
	}
	err := v.GetJSON(uri, &res)

	return res.Data.Attributes, err
}

func (v *API) Cockpit(accountID string, vin string) (Cockpit, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kca/car-adapter/v1/cars/%s/cockpit", v.keys.Target, accountID, vin)

	var res struct {
		Data struct {
			Attributes Cockpit `json:"attributes"`
		} `json:"data"`
	}
	err := v.GetJSON(uri, &res)

	return res.Data.Attributes, err
}

func (v *API) SocLevels(accountID string, vin string) (SocLevels, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kcm/v1/vehicles/%s/ev/soc-levels", v.keys.Target, accountID, vin)

	var res SocLevels
	err := v.GetJSON(uri, &res)

	return res, err
}

func (v *API) Position(accountID string, vin string) (Position, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kca/car-adapter/v1/cars/%s/location", v.keys.Target, accountID, vin)

	var res struct {
		Data struct {
			Attributes Position `json:"attributes"`
		} `json:"data"`
	}
	err := v.GetJSON(uri, &res)

	return res.Data.Attributes, err
}

func (v *API) DoAction(uri string, data any, res any) error {
	resContainer := map[string]any{
		"data": res,
	}

	reqData := map[string]any{
		"data": data,
	}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(reqData))

	if err != nil {
		return err
	}

	err = v.DoJSON(req, &resContainer)

	return err
}

func (v *API) WakeUp(accountID string, vin string) (ChargeAction, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kcm/v1/vehicles/%s/charge/pause-resume", v.keys.Target, accountID, vin)

	data := ChargeAction{
		Type: "ChargePauseResume",
		Attributes: ChargeActionAttributes{
			Action: "resume",
		},
	}

	var res ChargeAction
	err := v.DoAction(uri, data, &res)

	return res, err
}

func (v *API) WakeUpMy24(accountID string, vin string) (EvSettingsResponse, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kcm/v1/vehicles/%s/ev/settings", v.keys.Target, accountID, vin)

	data := EvSettingsRequest{
		LastSettingsUpdateTimestamp: "2025-04-24T12:41:41.823Z",
		DelegatedActivated:          false,
		ChargeModeRq:                "SCHEDULED",
		ChargeTimeStart:             "21:00",
		ChargeDuration:              1615,
		PreconditioningTemperature:  20.0,
		Programs:                    []any{},
	}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data))

	if err != nil {
		return EvSettingsResponse{}, err
	}

	var res EvSettingsResponse
	err = v.DoJSON(req, &res)

	return res, err
}

func (v *API) ChargeAction(accountID, action string, vin string) (ChargeAction, error) {
	uri := fmt.Sprintf("%s/commerce/v1/accounts/%s/kamereon/kca/car-adapter/v1/cars/%s/actions/charging-start", v.keys.Target, accountID, vin)

	data := ChargeAction{
		Type: "ChargingStart",
		Attributes: ChargeActionAttributes{
			Action: action,
		},
	}

	var res ChargeAction
	err := v.DoAction(uri, data, &res)

	return res, err
}

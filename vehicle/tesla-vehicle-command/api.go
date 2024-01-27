package vc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bogosj/tesla"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

const (
	FleetAudienceEU = "https://fleet-api.prd.eu.vn.cloud.tesla.com"

	CloudProxyURL = "https://tesla.evcc.io"
	TokenHeader   = "X-Authorization"
)

type API struct {
	*request.Helper
	identity *Identity
	base     string
}

func NewAPI(log *util.Logger, identity *Identity, timeout time.Duration) *API {
	client := request.NewHelper(log)
	client.Client.Timeout = timeout
	client.Transport = &oauth2.Transport{
		Source: identity,
		Base:   client.Transport,
	}

	return &API{
		Helper:   client,
		identity: identity,
		base:     FleetAudienceEU,
	}
}

func (v *API) Proxy(url, token string) {
	v.base = url

	v.Client.Transport = &transport.Decorator{
		Base: v.Client.Transport,
		Decorator: transport.DecorateHeaders(map[string]string{
			"X-Authorization": token,
		}),
	}
}

func (v *API) Region() (Region, error) {
	var res RegionResponse
	err := v.GetJSON(fmt.Sprintf("%s/api/1/users/region", FleetAudienceEU), &res)
	if err == nil {
		v.base = res.Response.FleetApiBaseUrl
	}
	return res.Response, err
}

func (v *API) Vehicles() ([]*Vehicle, error) {
	// ctx, cancel := context.WithTimeout(context.Background(), v.Timeout)
	// defer cancel()

	// b, err := v.identity.Account().Get(ctx, "api/1/vehicles")
	// if err != nil {
	// 	return nil, err
	// }

	// var res tesla.VehiclesResponse
	// if err := json.Unmarshal(b, &res); err != nil {
	// 	return nil, err
	// }

	var res tesla.VehiclesResponse
	err := v.GetJSON(fmt.Sprintf("%s/api/1/vehicles", v.base), &res)

	return res.Response, err
}

func (v *API) VehicleData(id int64) (*VehicleData, error) {
	// ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	// defer cancel()

	// b, err := v.identity.Account().Get(ctx, fmt.Sprintf("api/1/vehicles/%d/vehicle_data", id))
	// if err != nil {
	// 	return nil, err
	// }

	// var res tesla.VehicleData
	// if err := json.Unmarshal(b, &res); err != nil {
	// 	return nil, err
	// }

	var res tesla.VehicleData
	err := v.GetJSON(fmt.Sprintf("%s/api/1/vehicles/%d/vehicle_data", v.base, id), &res)

	return &res, err
}

func (v *API) WakeUp(id int64) (*VehicleData, error) {
	// ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	// defer cancel()

	// b, err := v.identity.Account().Get(ctx, fmt.Sprintf("api/1/vehicles/%d/vehicle_data", id))
	// if err != nil {
	// 	return nil, err
	// }

	// var res tesla.VehicleData
	// if err := json.Unmarshal(b, &res); err != nil {
	// 	return nil, err
	// }

	var res tesla.VehicleData
	err := v.GetJSON(fmt.Sprintf("%s/api/1/vehicles/%d/wake_up", v.base, id), &res)

	return &res, err
}

func (v *API) commandPath(vin, command string) string {
	basePath := strings.Join([]string{v.base, "vehicles", vin}, "/")
	return strings.Join([]string{basePath, "command", command}, "/")
}

// Sends a command to the vehicle
func (v *API) sendCommand(url string, reqBody []byte) ([]byte, error) {
	body, err := v.c.post(url, reqBody)
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		response := &CommandResponse{}
		if err := json.Unmarshal(body, response); err != nil {
			return nil, err
		}
		if !response.Response.Result && response.Response.Reason != "" {
			return nil, errors.New(response.Response.Reason)
		}
	}
	return body, nil
}

// SetChargingAmps set the charging amps to a specific value.
func (v *API) SetChargingAmps(vin string, amps int) error {
	apiURL := v.commandPath(vin, "set_charging_amps")
	payload := `{"charging_amps": ` + strconv.Itoa(amps) + `}`
	_, err := v.c.post(apiURL, []byte(payload))
	return err
}

// StartCharging starts the charging of the vehicle after you have inserted the charging cable.
func (v *API) StartCharging(vin string) error {
	apiURL := v.commandPath("charge_start")
	_, err := v.sendCommand(apiURL, nil)
	return err
}

// StopCharging stops the charging of the vehicle.
func (v *API) StopCharging(vin string) error {
	apiURL := v.commandPath("charge_stop")
	_, err := v.sendCommand(apiURL, nil)
	return err
}

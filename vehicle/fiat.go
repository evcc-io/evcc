package vehicle

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/fiat"
)

// https://github.com/TA2k/ioBroker.fiat

// Fiat is an api.Vehicle implementation for Fiat cars
type Fiat struct {
	*embed
	*request.Helper
	vin      string
	identity *fiat.Identity
	statusG  func() (interface{}, error)
}

func init() {
	registry.Add("fiat", NewFiatFromConfig)
}

// NewFiatFromConfig creates a new vehicle
func NewFiatFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, errors.New("missing credentials")
	}

	log := util.NewLogger("fiat")

	v := &Fiat{
		embed:    &cc.embed,
		Helper:   request.NewHelper(log),
		vin:      strings.ToUpper(cc.VIN),
		identity: fiat.NewIdentity(log, cc.User, cc.Password),
	}

	err := v.identity.Login()
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	if cc.VIN == "" {
		v.vin, err = findVehicle(v.vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.vin)
		}
	}

	v.statusG = provider.NewCached(func() (interface{}, error) {
		return v.status()
	}, cc.Cache).InterfaceGetter()

	return v, err
}

func (v *Fiat) request(method, uri string, body io.ReadSeeker) (*http.Request, error) {
	headers := map[string]string{
		"Content-Type":        "application/json",
		"X-Clientapp-Version": "1.0",
		"ClientrequestId":     util.RandomString(16),
		"X-Api-Key":           fiat.XApiKey,
		"X-Originator-Type":   "web",
	}

	req, err := request.New(method, uri, body, headers)
	if err == nil {
		err = v.identity.Sign(req, body)
	}

	return req, err
}

func (v *Fiat) vehicles() ([]string, error) {
	var res fiat.Vehicles

	uri := fmt.Sprintf("%s/v4/accounts/%s/vehicles?stage=ALL", fiat.ApiURI, v.identity.UID())

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

func (v *Fiat) status() (interface{}, error) {
	var res fiat.Status

	uri := fmt.Sprintf("%s/v2/accounts/%s/vehicles/%s/status", fiat.ApiURI, v.identity.UID(), v.vin)

	req, err := v.request(http.MethodGet, uri, nil)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return res, err
}

// SoC implements the api.Vehicle interface
func (v *Fiat) SoC() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(fiat.Status); err == nil && ok {
		return float64(res.EvInfo.Battery.StateOfCharge), nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Fiat)(nil)

// Range implements the api.VehicleRange interface
func (v *Fiat) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(fiat.Status); err == nil && ok {
		return int64(res.EvInfo.Battery.DistanceToEmpty.Value), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Fiat)(nil)

// Status implements the api.ChargeState interface
func (v *Fiat) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if res, ok := res.(fiat.Status); err == nil && ok {
		if res.EvInfo.Battery.PlugInStatus {
			status = api.StatusB // connected, not charging
		}
		if res.EvInfo.Battery.ChargingStatus == "CHARGING" {
			status = api.StatusC // charging
		}
	}

	return status, err
}

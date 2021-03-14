package vehicle

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"github.com/andig/evcc/vehicle/oidc"
)

const (
	fordAuth = "https://fcis.ice.ibmcloud.com"
	fordAPI  = "https://usapi.cv.ford.com"
)

// Ford is an api.Vehicle implementation for Ford cars
type Ford struct {
	*embed
	*request.Helper
	user, password, vin string
	tokens              oidc.Token
	chargeStateG        func() (float64, error)
}

func init() {
	registry.Add("ford", NewFordFromConfig)
}

// NewFordFromConfig creates a new vehicle
func NewFordFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title               string
		Capacity            int64
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("ford")

	v := &Ford{
		embed:    &embed{cc.Title, cc.Capacity},
		Helper:   request.NewHelper(log),
		user:     cc.User,
		password: cc.Password,
		vin:      strings.ToUpper(cc.VIN),
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	var err error
	if cc.VIN == "" {
		v.vin, err = findVehicle(v.vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.vin)
		}
	}

	return v, err
}

func (v *Ford) login(user, password string) error {
	data := url.Values{
		"client_id":  []string{"9fb503e0-715b-47e8-adfd-ad4b7770f73b"},
		"grant_type": []string{"password"},
		"username":   []string{user},
		"password":   []string{password},
	}

	uri := fordAuth + "/v1.0/endpoint/default/token"
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)
	if err != nil {
		return err
	}

	var tokens oidc.Token
	if err = v.DoJSON(req, &tokens); err == nil {
		v.tokens = tokens
	}

	return err
}

func (v *Ford) request(uri string) (*http.Request, error) {
	if v.tokens.AccessToken == "" || time.Until(v.tokens.Expiry) < time.Minute {
		if err := v.login(v.user, v.password); err != nil {
			return nil, err
		}
	}

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Content-type":   "application/json",
		"Application-Id": "71A3AD0A-CF46-4CCF-B473-FC7FE5BC4592",
		"Auth-Token":     v.tokens.AccessToken,
	})

	return req, err
}

// vehicles implements returns the list of user vehicles
func (v *Ford) vehicles() ([]string, error) {
	var resp struct {
		Vehicles struct {
			Values []struct {
				VIN string
			} `json:"$values"`
		}
	}

	var vehicles []string

	req, err := v.request("https://api.mps.ford.com/api/users/vehicles")
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	if err == nil {
		for _, v := range resp.Vehicles.Values {
			vehicles = append(vehicles, v.VIN)
		}
	}

	return vehicles, err
}

// chargeState implements the api.Vehicle interface
func (v *Ford) chargeState() (float64, error) {
	var resp struct {
		VehicleStatus struct {
			Battery struct {
				BatteryStatusActual struct {
					Value int
				}
			}
			BatteryFillLevel struct {
				Value float64
			}
		}

		// "chargingStatus": { "value": "EvseNotDetected", ...
		// "plugStatus": { "value": 1, ...
		// "batteryChargeStatus": null, ...
	}

	uri := fmt.Sprintf("%s/api/vehicles/v4/%s/status", fordAPI, v.vin)

	req, err := v.request(uri)
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	return resp.VehicleStatus.BatteryFillLevel.Value, err
}

// SoC implements the api.Vehicle interface
func (v *Ford) SoC() (float64, error) {
	return v.chargeStateG()
}

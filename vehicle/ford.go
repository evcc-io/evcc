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
	tokens              oidc.Tokens
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
	}{}

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
		// v.vin, err = findVehicle(v.vehicles())
		v.vin, err = findVehicle(nil, nil)
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", v.vin)
		}
	}

	return v, nil
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

	var tokens oidc.Tokens
	if err = v.DoJSON(req, &tokens); err == nil {
		tokens.Valid = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
		v.tokens = tokens
	}

	return err
}

func (v *Ford) request(uri string) (*http.Request, error) {
	if v.tokens.AccessToken == "" || time.Since(v.tokens.Valid) > 0 {
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

// // vehicles implements returns the list of user vehicles
// func (v *Ford) vehicles() ([]string, error) {
// 	var resp interface{}
// 	uri := fmt.Sprintf("%s/api/vehicles/v4/status", fordAPI)

// 	var vehicles []string

// 	req, err := v.request(uri)
// 	if err == nil {
// 		b, _ := httputil.DumpRequest(req, true)
// 		fmt.Println(string(b))
// 		err = v.DoJSON(req, &resp)
// 	}

// 	if err == nil {
// 		for _, v := range resp {
// 			vehicles = append(vehicles, v.VIN)
// 		}
// 	}

// 	// return vehicles, err
// 	return nil, err
// }

// chargeState implements the Vehicle.ChargeState interface
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

		// "chargingStatus": {
		// 	"value": "EvseNotDetected",
		// 	"status": "CURRENT",
		// 	"timestamp": "10-27-2020 12:53:01"
		// },
		// "plugStatus": {
		// 	"value": 1,
		// 	"status": "CURRENT",
		// 	"timestamp": "10-27-2020 12:53:01"
		// },
		// "batteryChargeStatus": null,
	}

	uri := fmt.Sprintf("%s/api/vehicles/v4/%s/status", fordAPI, v.vin)

	req, err := v.request(uri)
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	return resp.VehicleStatus.BatteryFillLevel.Value, err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Ford) ChargeState() (float64, error) {
	return v.chargeStateG()
}

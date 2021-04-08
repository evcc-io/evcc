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
	"github.com/andig/evcc/util/oauth"
	"github.com/andig/evcc/util/request"
)

const (
	fordAuth          = "https://fcis.ice.ibmcloud.com"
	fordAPI           = "https://usapi.cv.ford.com"
	fordVehicleList   = "https://api.mps.ford.com/api/users/vehicles"
	outdatedAfterMins = 5 // if returned status value is older than x minutes, evcc will init refresh
)

// Ford is an api.Vehicle implementation for Ford cars
type Ford struct {
	*embed
	*request.Helper
	log                 *util.Logger
	user, password, vin string
	tokens              oauth.Token
	chargeStateG        func() (float64, error)
	statusG             func() (interface{}, error)
}

/* Initialization of vehicle */

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
		log:      log,
		user:     cc.User,
		password: cc.Password,
		vin:      strings.ToUpper(cc.VIN),
	}

	v.statusG = provider.NewCached(func() (interface{}, error) {
		return v.VehicleStatus()
	}, cc.Cache).InterfaceGetter()

	// v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	var err error
	if cc.VIN == "" {
		v.vin, err = findVehicle(v.vehicles())
		if err == nil {
			v.log.DEBUG.Printf("found vehicle: %v", v.vin)
		}
	}

	return v, err
}

/* Authentication */

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

	var tokens oauth.Token
	if err = v.DoJSON(req, &tokens); err == nil {
		v.tokens = tokens
	}

	return err
}

/* Helper to send API requests */

func (v *Ford) request(method string, uri string) (*http.Request, error) {
	if v.tokens.AccessToken == "" || time.Until(v.tokens.Expiry) < time.Minute {
		if err := v.login(v.user, v.password); err != nil {
			return nil, err
		}
	}

	req, err := request.New(method, uri, nil, map[string]string{
		"Content-type":   "application/json",
		"Application-Id": "71A3AD0A-CF46-4CCF-B473-FC7FE5BC4592",
		"Auth-Token":     v.tokens.AccessToken,
	})

	return req, err
}

type vehicleStatus struct {
	VehicleStatus struct {
		BatteryFillLevel struct {
			Value     float64
			Timestamp string
		}
		ElVehDTE struct {
			Value     float64
			Timestamp string
		}
		ChargingStatus struct {
			Value     string
			Timestamp string
		}
		PlugStatus struct {
			Value     int
			Timestamp string
		}
		LastRefresh string
	}
	Status int
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

	req, err := v.request(http.MethodGet, fordVehicleList)
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

func (v *Ford) CalculateAge(timestamp string) (age time.Duration, err error) {
	var timestampTime time.Time
	const dateFormat = "01-02-2006 15:04:05"
	timestampTime, err = time.Parse(dateFormat, timestamp)
	age = time.Now().Sub(timestampTime)

	return age, err
}

// VehicleStatus implements the /status response
func (v *Ford) VehicleStatus() (res vehicleStatus, err error) {
	uri := fmt.Sprintf("%s/api/vehicles/v3/%s/status", fordAPI, v.vin)

	req, err := v.request(http.MethodGet, uri)
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	var statusAge time.Duration
	statusAge, err = v.CalculateAge(res.VehicleStatus.LastRefresh)
	v.log.DEBUG.Printf("Vehicle Status Age: %v", statusAge)
	// todo - Fehlerbehandlung, Timestamp-Format

	if statusAge > outdatedAfterMins*time.Minute {
		// received data is considered outdated, server is requested to poll updated data from vehicle
		v.log.DEBUG.Print("Vehicle Status is considered as outdated, requesting refresh")
		var updatedRes vehicleStatus
		updatedRes, err = v.VehicleStatusRefresh()
		if err == nil {
			res = updatedRes
			statusAge, err = v.CalculateAge(res.VehicleStatus.LastRefresh)
			v.log.DEBUG.Printf("Refreshed Status Age: %v", statusAge)
		}
	}

	return res, err
}

// Status implements the /status response
func (v *Ford) VehicleStatusRefresh() (res vehicleStatus, err error) {
	var commandId string
	commandId, err = v.requestRefresh()

	if err == nil {
		uri := fmt.Sprintf("%s/api/vehicles/v3/%s/statusrefresh/%s", fordAPI, v.vin, commandId)

		var req *http.Request

		counter := 0
		const maxTrials = 20
		for counter < maxTrials {
			req, err = v.request(http.MethodGet, uri)
			if err == nil {
				err = v.DoJSON(req, &res)
			} else {
				break
			}

			// if status = 200, the update is complete
			if res.Status == 200 {
				break
			}

			v.log.TRACE.Printf("Status of data refresh: %v", res.Status)

			time.Sleep(1500 * time.Millisecond)
			counter++
		}

		if counter >= maxTrials && res.Status != 200 {
			err = fmt.Errorf("update of SoC not completed after timeout")
			v.log.DEBUG.Print("update of SoC not completed after timeout")
		}
	}

	return res, err
}

// Request API to poll vehicle for updated data
// returns commandId to get the result after server received data from vehicle
func (v *Ford) requestRefresh() (string, error) {
	var resp struct {
		CommandId string
	}

	uri := fmt.Sprintf("%s/api/vehicles/v2/%s/status", fordAPI, v.vin)
	req, err := v.request(http.MethodPut, uri)
	if err == nil {
		err = v.DoJSON(req, &resp)
	}

	return resp.CommandId, err
}

var _ api.Battery = (*Ford)(nil)

// SoC implements the api.Vehicle interface
func (v *Ford) SoC() (float64, error) {
	res, err := v.statusG()
	if res, ok := res.(vehicleStatus); err == nil && ok {
		return float64(res.VehicleStatus.BatteryFillLevel.Value), nil
	}

	return 0, err
}

var _ api.VehicleRange = (*Ford)(nil)

// Range implements the api.VehicleRange interface
func (v *Ford) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(vehicleStatus); err == nil && ok {
		return int64(res.VehicleStatus.ElVehDTE.Value), nil
	}

	return 0, err
}

var _ api.ChargeState = (*Ford)(nil)

// Status implements the api.ChargeState interface
func (v *Ford) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if res, ok := res.(vehicleStatus); err == nil && ok {
		if res.VehicleStatus.PlugStatus.Value == 1 {
			status = api.StatusB
		}
		if res.VehicleStatus.ChargingStatus.Value == "ChargingAC" {
			status = api.StatusC
		}
	}

	return status, err
}

package charger

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

const (
	evseGetParameters apiFunction = "getParameters"
	evseSetStatus     apiFunction = "setStatus"
	evseSetCurrent    apiFunction = "setCurrent"

	evseSuccess = "S0_"
)

// EVSEParameterResponse is the getParameters response
type EVSEParameterResponse struct {
	Type string          `json:"type"`
	List []EVSEListEntry `json:"list"`
}

// EVSEListEntry is EVSEParameterResponse.List
type EVSEListEntry struct {
	VehicleState   int64   `json:"vehicleState"`
	EvseState      bool    `json:"evseState"`
	MaxCurrent     int64   `json:"maxCurrent"`
	ActualCurrent  int64   `json:"actualCurrent"`
	ActualPower    float64 `json:"actualPower"`
	Duration       int64   `json:"duration"`
	AlwaysActive   bool    `json:"alwaysActive"`
	LastActionUser string  `json:"lastActionUser"`
	LastActionUID  string  `json:"lastActionUID"`
	Energy         float64 `json:"energy"`
	Mileage        float64 `json:"mileage"`
	MeterReading   float64 `json:"meterReading"`
	CurrentP1      float64 `json:"currentP1"`
	CurrentP2      float64 `json:"currentP2"`
	CurrentP3      float64 `json:"currentP3"`
}

// EVSEWifi charger implementation
type EVSEWifi struct {
	*util.HTTPHelper
	uri string
	alwaysActive bool
}

func init() {
	registry.Add("evsewifi", NewEVSEWifiFromConfig)
}

// NewEVSEWifiFromConfig creates a EVSEWifi charger from generic config
func NewEVSEWifiFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct{ URI string }{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEVSEWifi(cc.URI)
}

// NewEVSEWifi creates EVSEWifi charger
func NewEVSEWifi(uri string) (api.Charger, error) {
	evse := &EVSEWifi{
		HTTPHelper:        util.NewHTTPHelper(util.NewLogger("wifi")),
		uri:               strings.TrimRight(uri, "/"),
		alwaysActive:      true,
	}

	return evse, nil
}

func (evse *EVSEWifi) apiURL(service apiFunction) string {
	return fmt.Sprintf("%s/%s", evse.uri, service)
}

// query evse parameters
func (evse *EVSEWifi) getParameters() (EVSEListEntry, error) {
	var pr EVSEParameterResponse
	url := evse.apiURL(evseGetParameters)
	body, err := evse.GetJSON(url, &pr)
	if err != nil {
		return EVSEListEntry{}, err
	}

	if len(pr.List) != 1 {
		return EVSEListEntry{}, fmt.Errorf("unexpected response: %s", string(body))
	}

	params := pr.List[0]
	if !params.AlwaysActive {
		evse.HTTPHelper.Log.WARN.Println("evse should be configured to remote mode")
	}

	evse.alwaysActive = params.AlwaysActive
	return params, nil
}

// Status implements the Charger.Status interface
func (evse *EVSEWifi) Status() (api.ChargeStatus, error) {
	params, err := evse.getParameters()
	if err != nil {
		return api.StatusNone, err
	}

	switch params.VehicleState {
	case 1: // ready
		return api.StatusA, nil
	case 2: // EV is present
		return api.StatusB, nil
	case 3: // charging
		return api.StatusC, nil
	case 4: // charging with ventilation
		return api.StatusD, nil
	case 5: // failure (e.g. diode check, RCD failure)
		return api.StatusE, nil
	default:
		return api.StatusNone, errors.New("invalid response")
	}
}

// Enabled implements the Charger.Enabled interface
func (evse *EVSEWifi) Enabled() (bool, error) {
	params, err := evse.getParameters()
	return params.EvseState, err
}

// checkError checks for EVSE error response with HTTP 200 status
func (evse *EVSEWifi) checkError(b []byte, err error) error {
	if err == nil && !strings.HasPrefix(string(b), evseSuccess) {
		err = errors.New(string(b))
	}
	return err
}

// Enable implements the Charger.Enable interface
func (evse *EVSEWifi) Enable(enable bool) error {
	url := fmt.Sprintf("%s?active=%v", evse.apiURL(evseSetStatus), enable)

	if evse.alwaysActive {
		current := 0
		if enable {
			current = 6
		}
		url = fmt.Sprintf("%s?current=%d", evse.apiURL(evseSetCurrent), current)
	}
	return evse.checkError(evse.Get(url))
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (evse *EVSEWifi) MaxCurrent(current int64) error {
	url := fmt.Sprintf("%s?current=%d", evse.apiURL(evseSetCurrent), current)
	return evse.checkError(evse.Get(url))
}

// ChargingTime yields current charge run duration
func (evse *EVSEWifi) ChargingTime() (time.Duration, error) {
	params, err := evse.getParameters()
	return time.Duration(params.Duration) * time.Millisecond, err
}

// // TotalEnergy implements the MeterEnergy interface
// func (evse *EVSEWifi) TotalEnergy() (float64, error) {
// 	params, err := evse.getParameters()
// 	return params.MeterReading, err
// }

// // ChargedEnergy implements the ChargeRater interface
// func (evse *EVSEWifi) ChargedEnergy() (float64, error) {
// 	params, err := evse.getParameters()
// 	return params.Energy, err
// }

// // Currents implements the MeterCurrents interface
// func (evse *EVSEWifi) Currents() (float64, float64, float64, error) {
// 	params, err := evse.getParameters()
// 	return float64(params.CurrentP1), float64(params.CurrentP2), float64(params.CurrentP3), err
// }

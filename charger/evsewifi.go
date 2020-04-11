package charger

import (
	"errors"
	"fmt"
	"strings"

	"github.com/andig/evcc/api"
)

const (
	evseGetParameters apiFunction = "getParameters"
	evseSetStatus     apiFunction = "setStatus"
	evseSetCurrent    apiFunction = "setCurrent"

	evseSuccess = "S0_EVSE"
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
	*api.HTTPHelper
	uri string
}

// NewEVSEWifiFromConfig creates a EVSEWifi charger from generic config
func NewEVSEWifiFromConfig(log *api.Logger, other map[string]interface{}) api.Charger {
	cc := struct{ URI string }{}
	api.DecodeOther(log, other, &cc)

	return NewEVSEWifi(cc.URI)
}

// NewEVSEWifi creates EVSEWifi charger
func NewEVSEWifi(uri string) api.Charger {
	evse := &EVSEWifi{
		HTTPHelper: api.NewHTTPHelper(api.NewLogger("wifi")),
		uri:        strings.TrimRight(uri, "/") + "/",
	}

	evse.HTTPHelper.Log.WARN.Println("-- experimental --")

	return evse
}

func (evse *EVSEWifi) apiURL(service apiFunction) string {
	return fmt.Sprintf("%s/%s", evse.uri, service)
}

// Status implements the Charger.Status interface
func (evse *EVSEWifi) Status() (api.ChargeStatus, error) {
	var pr EVSEParameterResponse
	url := evse.apiURL(evseGetParameters)
	body, err := evse.GetJSON(url, &pr)
	if err != nil {
		return api.StatusNone, err
	}

	if len(pr.List) != 1 {
		return api.StatusNone, fmt.Errorf("unexpected response: %s", string(body))
	}

	switch pr.List[0].VehicleState {
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
	var pr EVSEParameterResponse
	url := evse.apiURL(evseGetParameters)
	body, err := evse.GetJSON(url, &pr)
	if err != nil {
		return false, err
	}

	if len(pr.List) != 1 {
		return false, fmt.Errorf("unexpected response: %s", string(body))
	}

	return pr.List[0].EvseState, nil
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
	return evse.checkError(evse.Get(url))
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (evse *EVSEWifi) MaxCurrent(current int64) error {
	url := fmt.Sprintf("%s?current=%d", evse.apiURL(evseSetCurrent), current)
	return evse.checkError(evse.Get(url))
}

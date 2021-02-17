package charger

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

const (
	evseGetParameters = "getParameters"
	evseSetStatus     = "setStatus"
	evseSetCurrent    = "setCurrent"

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
	UseMeter       bool    `json:"useMeter"`
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
	*request.Helper
	log          *util.Logger
	uri          string
	alwaysActive bool
	current      int64
}

func init() {
	registry.Add("evsewifi", NewEVSEWifiFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -p charger -f decorateEVSE -o evsewifi_decorators -b *EVSEWifi -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)"

// NewEVSEWifiFromConfig creates a EVSEWifi charger from generic config
func NewEVSEWifiFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI   string
		Meter struct {
			Power, Energy, Currents bool
		}
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	evse, err := NewEVSEWifi(util.DefaultScheme(cc.URI, "http"))
	if err != nil {
		return evse, err
	}

	// auto-detect EVSE meter
	if meter, err := evse.HasMeter(); meter && err == nil {
		cc.Meter.Energy = true
		cc.Meter.Energy = true
		cc.Meter.Currents = true
	}

	// decorate Charger with Meter
	var currentPower func() (float64, error)
	if cc.Meter.Energy {
		currentPower = evse.currentPower
	}

	// decorate Charger with MeterEnergy
	var totalEnergy func() (float64, error)
	if cc.Meter.Energy {
		totalEnergy = evse.totalEnergy
	}

	// decorate Charger with MeterCurrent
	var currents func() (float64, float64, float64, error)
	if cc.Meter.Currents {
		currents = evse.currents
	}

	return decorateEVSE(evse, currentPower, totalEnergy, currents), nil
}

// NewEVSEWifi creates EVSEWifi charger
func NewEVSEWifi(uri string) (*EVSEWifi, error) {
	log := util.NewLogger("evse")

	evse := &EVSEWifi{
		log:     log,
		Helper:  request.NewHelper(log),
		uri:     strings.TrimRight(uri, "/"),
		current: 6, // 6A defined value
	}

	return evse, nil
}

func (evse *EVSEWifi) apiURL(service string) string {
	return fmt.Sprintf("%s/%s", evse.uri, service)
}

// query evse parameters
func (evse *EVSEWifi) getParameters() (EVSEListEntry, error) {
	var res EVSEParameterResponse
	url := evse.apiURL(evseGetParameters)
	err := evse.GetJSON(url, &res)
	if err != nil {
		return EVSEListEntry{}, err
	}

	if len(res.List) != 1 {
		return EVSEListEntry{}, fmt.Errorf("unexpected response: %s", res.Type)
	}

	params := res.List[0]
	if !params.AlwaysActive {
		evse.log.WARN.Println("evse should be configured to remote mode")
	}

	evse.alwaysActive = params.AlwaysActive
	return params, nil
}

// HasMeter returns the useMeter api response
func (evse *EVSEWifi) HasMeter() (bool, error) {
	params, err := evse.getParameters()
	return params.UseMeter, err
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
		var current int64
		if enable {
			current = evse.current
		}
		url = fmt.Sprintf("%s?current=%d", evse.apiURL(evseSetCurrent), current)
	}
	return evse.checkError(evse.GetBody(url))
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (evse *EVSEWifi) MaxCurrent(current int64) error {
	evse.current = current
	url := fmt.Sprintf("%s?current=%d", evse.apiURL(evseSetCurrent), current)
	return evse.checkError(evse.GetBody(url))
}

// ChargingTime yields current charge run duration
func (evse *EVSEWifi) ChargingTime() (time.Duration, error) {
	params, err := evse.getParameters()
	return time.Duration(params.Duration) * time.Millisecond, err
}

// CurrentPower implements the Meter interface
func (evse *EVSEWifi) currentPower() (float64, error) {
	params, err := evse.getParameters()
	return 1000 * params.ActualPower, err
}

// TotalEnergy implements the MeterEnergy interface
func (evse *EVSEWifi) totalEnergy() (float64, error) {
	params, err := evse.getParameters()
	return params.MeterReading, err
}

// Currents implements the MeterCurrents interface
func (evse *EVSEWifi) currents() (float64, float64, float64, error) {
	params, err := evse.getParameters()
	return float64(params.CurrentP1), float64(params.CurrentP2), float64(params.CurrentP3), err
}

// // ChargedEnergy implements the ChargeRater interface
// func (evse *EVSEWifi) ChargedEnergy() (float64, error) {
// 	params, err := evse.getParameters()
// 	return params.Energy, err
// }

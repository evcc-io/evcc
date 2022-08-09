package charger

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/evse"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// EVSEWifi charger implementation
type EVSEWifi struct {
	*request.Helper
	uri          string
	alwaysActive bool
	current      int64 // current will always be the physical value sent to the API
	hires        bool
}

func init() {
	registry.Add("smartwb", NewEVSEWifiFromConfig)
	registry.Add("evsewifi", NewEVSEWifiFromConfig)
}

// go:generate go run ../cmd/tools/decorate.go -f decorateEVSE -b *EVSEWifi -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)" -t "api.ChargerEx,MaxCurrentMillis,func(current float64) error" -t "api.Identifier,Identify,func() (string, error)"

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

	wb, err := NewEVSEWifi(util.DefaultScheme(cc.URI, "http"))
	if err != nil {
		return wb, err
	}

	// auto-detect capabilities
	params, err := wb.getParameters()
	if err != nil {
		return wb, err
	}

	if !params.AlwaysActive {
		return nil, errors.New("evse must be configured to remote mode")
	}

	if params.UseMeter {
		cc.Meter.Energy = true
		cc.Meter.Energy = true
		cc.Meter.Currents = true
	}

	if params.ActualCurrentMA != nil {
		wb.hires = true
	}

	// decorate Charger with Meter
	var currentPower func() (float64, error)
	if cc.Meter.Energy {
		currentPower = wb.currentPower
	}

	// decorate Charger with MeterEnergy
	var totalEnergy func() (float64, error)
	if cc.Meter.Energy {
		totalEnergy = wb.totalEnergy
	}

	// decorate Charger with MeterCurrent
	var currents func() (float64, float64, float64, error)
	if cc.Meter.Currents {
		currents = wb.currents
	}

	// decorate Charger with MaxCurrentEx
	var maxCurrentEx func(float64) error
	if wb.hires {
		maxCurrentEx = wb.maxCurrentEx
		wb.current = 100 * wb.current
	}

	// decorate Charger with Identifier
	var identify func() (string, error)
	if params.RFIDUID != nil {
		identify = wb.identify
	}

	return decorateEVSE(wb, currentPower, totalEnergy, currents, maxCurrentEx, identify), nil
}

// NewEVSEWifi creates EVSEWifi charger
func NewEVSEWifi(uri string) (*EVSEWifi, error) {
	log := util.NewLogger("evse")

	wb := &EVSEWifi{
		Helper:  request.NewHelper(log),
		uri:     strings.TrimRight(uri, "/"),
		current: 6, // 6A defined value
	}

	return wb, nil
}

// query evse parameters
func (wb *EVSEWifi) getParameters() (evse.ListEntry, error) {
	var res evse.ParameterResponse
	uri := fmt.Sprintf("%s/getParameters", wb.uri)
	err := wb.GetJSON(uri, &res)
	if err != nil {
		return evse.ListEntry{}, err
	}

	if len(res.List) != 1 {
		return evse.ListEntry{}, fmt.Errorf("unexpected response: %s", res.Type)
	}

	params := res.List[0]
	wb.alwaysActive = params.AlwaysActive

	return params, nil
}

// Status implements the api.Charger interface
func (wb *EVSEWifi) Status() (api.ChargeStatus, error) {
	params, err := wb.getParameters()
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

// Enabled implements the api.Charger interface
func (wb *EVSEWifi) Enabled() (bool, error) {
	params, err := wb.getParameters()
	return params.EvseState, err
}

// get executes GET request and checks for EVSE error response
func (wb *EVSEWifi) get(uri string) error {
	b, err := wb.GetBody(uri)
	if err == nil && !strings.HasPrefix(string(b), evse.Success) {
		err = errors.New(string(b))
	}
	return err
}

// Enable implements the api.Charger interface
func (wb *EVSEWifi) Enable(enable bool) error {
	uri := fmt.Sprintf("%s/setStatus?active=%v", wb.uri, enable)
	if wb.alwaysActive {
		var current int64
		if enable {
			current = wb.current
		}
		uri = fmt.Sprintf("%s/setCurrent?current=%d", wb.uri, current)
	}
	return wb.get(uri)
}

// MaxCurrent implements the api.Charger interface
func (wb *EVSEWifi) MaxCurrent(current int64) error {
	if wb.hires {
		current = 100 * current
	}
	wb.current = current
	uri := fmt.Sprintf("%s/setCurrent?current=%d", wb.uri, current)
	return wb.get(uri)
}

// maxCurrentEx implements the api.ChargerEx interface
func (wb *EVSEWifi) maxCurrentEx(current float64) error {
	wb.current = int64(100 * current)
	uri := fmt.Sprintf("%s/setCurrent?current=%d", wb.uri, wb.current)
	return wb.get(uri)
}

var _ api.ChargeTimer = (*EVSEWifi)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *EVSEWifi) ChargingTime() (time.Duration, error) {
	params, err := wb.getParameters()
	return time.Duration(params.Duration) * time.Millisecond, err
}

// CurrentPower implements the api.Meter interface
func (wb *EVSEWifi) currentPower() (float64, error) {
	params, err := wb.getParameters()
	return 1000 * params.ActualPower, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *EVSEWifi) totalEnergy() (float64, error) {
	params, err := wb.getParameters()
	return params.MeterReading, err
}

// Currents implements the api.MeterCurrents interface
func (wb *EVSEWifi) currents() (float64, float64, float64, error) {
	params, err := wb.getParameters()
	return params.CurrentP1, params.CurrentP2, params.CurrentP3, err
}

// Identify implements the api.Identifier interface
func (wb *EVSEWifi) identify() (string, error) {
	params, err := wb.getParameters()
	if err != nil {
		return "", err
	}

	// we can rely on RFIDUID != nil here since identify() is only exposed if the EVSE API supports that property
	return *params.RFIDUID, nil
}

// // ChargedEnergy implements the ChargeRater interface
// func (wb *EVSEWifi) ChargedEnergy() (float64, error) {
// 	params, err := wb.getParameters()
// 	return params.Energy, err
// }

var _ api.Resurrector = (*EVSEWifi)(nil)

// WakeUp implements the Resurrector interface
func (wb *EVSEWifi) WakeUp() error {
	uri := fmt.Sprintf("%s/interruptCp", wb.uri)
	_, err := wb.GetBody(uri)
	return err
}

package charger

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Twc3 is an api.Vehicle implementation for Twc3 cars
type Twc3 struct {
	*request.Helper
	log     *util.Logger
	lp      loadpoint.API
	uri     string
	vitalsG func() (Vitals, error)
	enabled bool
}

func init() {
	registry.Add("twc3", NewTwc3FromConfig)
}

// Vitals is the /api/1/vitals response
type Vitals struct {
	ContactorClosed   bool    `json:"contactor_closed"`    // false
	VehicleConnected  bool    `json:"vehicle_connected"`   // false
	SessionS          int64   `json:"session_s"`           // 0
	GridV             float64 `json:"grid_v"`              // 230.1
	GridHz            float64 `json:"grid_hz"`             // 49.928
	VehicleCurrentA   float64 `json:"vehicle_current_a"`   // 0.1
	CurrentAA         float64 `json:"currentA_a"`          // 0.0
	CurrentBA         float64 `json:"currentB_a"`          // 0.1
	CurrentCA         float64 `json:"currentC_a"`          // 0.0
	CurrentNA         float64 `json:"currentN_a"`          // 0.0
	VoltageAV         float64 `json:"voltageA_v"`          // 0.0
	VoltageBV         float64 `json:"voltageB_v"`          // 0.0
	VoltageCV         float64 `json:"voltageC_v"`          // 0.0
	RelayCoilV        float64 `json:"relay_coil_v"`        // 11.8
	PcbaTempC         float64 `json:"pcba_temp_c"`         // 19.2
	HandleTempC       float64 `json:"handle_temp_c"`       // 15.3
	McuTempC          float64 `json:"mcu_temp_c"`          // 25.1
	UptimeS           int     `json:"uptime_s"`            // 831580
	InputThermopileUv float64 `json:"input_thermopile_uv"` //-233
	ProxV             float64 `json:"prox_v"`              // 0.0
	PilotHighV        float64 `json:"pilot_high_v"`        // 11.9
	PilotLowV         float64 `json:"pilot_low_v"`         // 11.9
	SessionEnergyWh   float64 `json:"session_energy_wh"`   // 22864.699
	ConfigStatus      int     `json:"config_status"`       // 5
	EvseState         int     `json:"evse_state"`          // 1
	CurrentAlerts     []any   `json:"current_alerts"`      //[]
}

// NewTwc3FromConfig creates a new vehicle
func NewTwc3FromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI   string
		Cache time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("twc3")

	c := &Twc3{
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(strings.TrimSuffix(cc.URI, "/"), "http"),
		log:    log,
	}

	c.vitalsG = provider.Cached(func() (Vitals, error) {
		var res Vitals
		uri := fmt.Sprintf("%s/api/1/vitals", c.uri)
		err := c.GetJSON(uri, &res)
		return res, err
	}, time.Second)

	return c, nil
}

// Status implements the api.Charger interface
func (v *Twc3) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.vitalsG()
	switch {
	case res.ContactorClosed:
		status = api.StatusC
	case res.VehicleConnected:
		status = api.StatusB
	}

	return status, err
}

// Enabled implements the api.Charger interface
func (c *Twc3) Enabled() (bool, error) {
	return verifyEnabled(c, c.enabled)
}

// Enable implements the api.Charger interface
func (c *Twc3) Enable(enable bool) error {
	if c.lp == nil {
		return errors.New("loadpoint not initialized")
	}

	// ignore disabling when vehicle is already disconnected
	// https://github.com/evcc-io/evcc/issues/10213
	status, err := c.Status()
	if err != nil {
		return err
	}
	if status == api.StatusA && !enable {
		c.enabled = false
		return nil
	}

	v, ok := c.lp.GetVehicle().(api.VehicleChargeController)
	if !ok {
		return errors.New("vehicle not capable of start/stop")
	}

	if enable {
		err = v.StartCharge()
	} else {
		err = v.StopCharge()
	}

	if err == nil {
		c.enabled = enable
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *Twc3) MaxCurrent(current int64) error {
	if c.lp == nil {
		return errors.New("loadpoint not initialized")
	}

	v, ok := c.lp.GetVehicle().(api.CurrentLimiter)
	if !ok {
		return errors.New("vehicle not capable of current control")
	}

	return v.MaxCurrent(current)
}

var _ api.ChargeRater = (*Twc3)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (v *Twc3) ChargedEnergy() (float64, error) {
	res, err := v.vitalsG()
	return res.SessionEnergyWh / 1e3, err
}

var _ api.ChargeTimer = (*Twc3)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (v *Twc3) ChargingTime() (time.Duration, error) {
	res, err := v.vitalsG()
	return time.Duration(res.SessionS) * time.Second, err
}

// Use workaround if voltageC_v is approximately half of grid_v
//
//	"voltageA_v": 241.5,
//	"voltageB_v": 241.5,
//	"voltageC_v": 118.7,
//
// Default state is ~2V on all phases unless charging
func (v *Twc3) isSplitPhase(res Vitals) bool {
	return math.Abs(res.VoltageCV-res.GridV/2) < 25
}

var _ api.PhaseCurrents = (*Twc3)(nil)

// Currents implements the api.PhaseCurrents interface
func (v *Twc3) Currents() (float64, float64, float64, error) {
	res, err := v.vitalsG()
	if v.isSplitPhase(res) {
		return res.CurrentAA + res.CurrentBA, 0, 0, err
	}
	return res.CurrentAA, res.CurrentBA, res.CurrentCA, err
}

var _ api.Meter = (*Twc3)(nil)

// CurrentPower implements the api.Meter interface
func (v *Twc3) CurrentPower() (float64, error) {
	res, err := v.vitalsG()
	if res.ContactorClosed {
		if v.isSplitPhase(res) {
			return (res.CurrentAA * res.VoltageAV) + (res.CurrentBA * res.VoltageBV), err
		}
		return (res.CurrentAA * res.VoltageAV) + (res.CurrentBA * res.VoltageBV) + (res.CurrentCA * res.VoltageCV), err
	}
	return 0, err
}

var _ api.PhaseVoltages = (*Twc3)(nil)

// Voltages implements the api.PhaseVoltages interface
func (v *Twc3) Voltages() (float64, float64, float64, error) {
	res, err := v.vitalsG()
	if v.isSplitPhase(res) {
		return (res.VoltageAV + res.VoltageBV) / 2, 0, 0, err
	}
	return res.VoltageAV, res.VoltageBV, res.VoltageCV, err
}

var _ loadpoint.Controller = (*Twc3)(nil)

// LoadpointControl implements loadpoint.Controller
func (v *Twc3) LoadpointControl(lp loadpoint.API) {
	v.lp = lp
}

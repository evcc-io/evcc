package charger

import (
	"errors"
	"math"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Twc3 is an api.Charger implementation for the Tesla Wall Connector Gen 3
type Twc3 struct {
	lp              loadpoint.API
	vitalsG         func() (Vitals, error)
	enabled         bool
	fleet           *twc3Fleet // nil without tesla block
	switchCurrent   float64    // on/off threshold current, only used in switch mode
	lockMaybeActive bool       // schedule may be gating the contactor (true = unknown/locked)
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
	CurrentAlerts     []any   `json:"current_alerts"`      // []
}

// NewTwc3FromConfig creates a new charger
func NewTwc3FromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		URI           string
		Cache         time.Duration
		SwitchCurrent float64          // on/off current override; 0 = read from wall connector
		Tesla         *twc3TeslaConfig // optional vehicle-independent on/off
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	c := &Twc3{
		lockMaybeActive: true, // unknown at startup -> first enable clears any stale lock once
	}

	log := util.NewLogger("twc3")
	client := request.NewHelper(log)
	baseURI := util.DefaultScheme(strings.TrimSuffix(cc.URI, "/"), "http")

	c.vitalsG = util.Cached(func() (Vitals, error) {
		var res Vitals
		err := client.GetJSON(baseURI+"/api/1/vitals", &res)
		return res, err
	}, cc.Cache)

	// optional vehicle-independent on/off via Tesla Fleet API
	if cc.Tesla != nil {
		fleet, switchCurrent, err := newTwc3Fleet(log, client, baseURI, cc.Tesla, cc.SwitchCurrent)
		if err != nil {
			return nil, err
		}
		c.fleet = fleet
		c.switchCurrent = switchCurrent
	}

	return c, nil
}

// switchMode reports whether the on/off fallback is in effect:
// fleet configured AND the connected vehicle cannot control current.
func (c *Twc3) switchMode() bool {
	return c.fleet != nil && c.lp != nil && !api.HasCap[api.CurrentController](c.lp.GetVehicle())
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
		return ErrLoadpointNotInitialized
	}

	// ignore disabling when the vehicle is already disconnected
	// https://github.com/evcc-io/evcc/issues/10213
	status, err := c.Status()
	if err != nil {
		return err
	}
	if status == api.StatusA && !enable {
		c.enabled = false
		return nil
	}

	if err := c.setCharging(enable); err != nil {
		return err
	}
	c.enabled = enable
	return nil
}

// setCharging starts/stops charging via the vehicle when it can, otherwise via the
// wall connector schedule (guest / non-Tesla / no vehicle).
func (c *Twc3) setCharging(enable bool) error {
	// 1. vehicle can start/stop (Tesla) -> control via vehicle
	if v, ok := api.Cap[api.ChargeController](c.lp.GetVehicle()); ok {
		// a guest may have locked the contactor via the schedule; clear it before
		// handing control back to the vehicle, but only when a lock may actually be
		// active - avoids a redundant Fleet API call on every enable. Disabling needs
		// no schedule change - the vehicle stops charging itself.
		if c.fleet != nil && enable && c.lockMaybeActive {
			if err := c.fleet.switchSchedule(true); err != nil {
				return err
			}
			c.lockMaybeActive = false
		}
		return v.ChargeEnable(enable)
	}

	// 2. fallback: vehicle cannot -> wall connector schedule
	if c.fleet != nil {
		if err := c.fleet.switchSchedule(enable); err != nil {
			return err
		}
		// the schedule now gates the contactor iff we disabled charging
		c.lockMaybeActive = !enable
		return nil
	}

	// 3. neither
	return errors.New("vehicle not capable of start/stop and no wall connector fallback configured")
}

// MaxCurrent implements the api.Charger interface
func (c *Twc3) MaxCurrent(current int64) error {
	if c.lp == nil {
		return ErrLoadpointNotInitialized
	}

	v, ok := api.Cap[api.CurrentController](c.lp.GetVehicle())
	if !ok {
		// hardware cannot set amps; on/off full load. Tolerate in all modes.
		return nil
	}

	return v.MaxCurrent(current)
}

var _ api.CurrentLimiter = (*Twc3)(nil)

// GetMinMaxCurrent implements the api.CurrentLimiter interface
func (c *Twc3) GetMinMaxCurrent() (float64, float64, error) {
	if c.switchMode() && c.switchCurrent > 0 {
		// min=max -> on/off only at full power
		return c.switchCurrent, c.switchCurrent, nil
	}
	// Tesla/classic, or unknown switch current: vehicle/loadpoint range applies
	return 0, 0, api.ErrNotAvailable
}

var _ api.CurrentGetter = (*Twc3)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (c *Twc3) GetMaxCurrent() (float64, error) {
	if c.lp == nil {
		return 0, ErrLoadpointNotInitialized
	}

	v, ok := api.Cap[api.CurrentGetter](c.lp.GetVehicle())
	if !ok {
		return 0, api.ErrNotAvailable
	}

	return v.GetMaxCurrent()
}

var _ api.ChargeRater = (*Twc3)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (v *Twc3) ChargedEnergy() (float64, error) {
	res, err := v.vitalsG()
	return res.SessionEnergyWh / 1e3, err
}

var _ api.ConnectionTimer = (*Twc3)(nil)

// ConnectionDuration implements the api.ConnectionTimer interface
func (v *Twc3) ConnectionDuration() (time.Duration, error) {
	res, err := v.vitalsG()
	return time.Duration(res.SessionS) * time.Second, err
}

// removed: https://github.com/evcc-io/evcc/issues/13555
// var _ api.ChargeTimer = (*Twc3)(nil)

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

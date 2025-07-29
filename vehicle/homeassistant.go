// SPDX-License-Identifier: MIT
// homeassistant.go – Vehicle adapter for evcc
//
// Integration of vehicle data from Home Assistant including
// charge control (start/stop), target SOC, odometer, climatisation, max current, finish time, wakeup, etc. via sensors and script services.
//
// YAML example:
// vehicles:
//   - name: id4
//     type: homeassistant
//     host: http://ha.local:8123
//     token: !secret ha_token
//     sensors:
//       soc: sensor.id4_soc                # State of charge [%] (required)
//       range: sensor.id4_range            # Remaining range [km] (optional)
//       status: sensor.id4_charging        # Charging status (optional)
//       limitSoc: number.id4_target_soc    # Target state of charge [%] (optional)
//       odometer: sensor.id4_odometer      # Odometer [km] (optional)
//       climater: binary_sensor.id4_clima  # Climatisation active (optional)
//       maxCurrent: sensor.id4_max_current # Max current [A] (optional)
//       getMaxCurrent: sensor.id4_actual_max_current # Actual max current [A] (optional)
//       finishTime: sensor.id4_finish_time # Finish time (ISO8601 or Unix, optional)
//     services:
//       start_charging: script.id4_start   # Start charging (optional)
//       stop_charging:  script.id4_stop    # Stop charging (optional)
//       wakeup: script.id4_wakeup          # Wake up vehicle (optional)
//     capacity: 77
//
// Notes:
// - All sensors must exist as entities in Home Assistant and provide a suitable value.
// - Changes to the vehicle API are handled exclusively in Home Assistant scripts/sensors.
// - Unused fields can be omitted.

package vehicle

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// -----------------------------
// Configuration structures
// -----------------------------

type haSensors struct {
	Soc           string `mapstructure:"soc"`           // z.B. sensor.id4_soc  (erforderlich)
	Range         string `mapstructure:"range"`         // optional
	Status        string `mapstructure:"status"`        // optional
	LimitSoc      string `mapstructure:"limitSoc"`      // optional
	Odometer      string `mapstructure:"odometer"`      // optional
	Climater      string `mapstructure:"climater"`      // optional
	MaxCurrent    string `mapstructure:"maxCurrent"`    // optional
	GetMaxCurrent string `mapstructure:"getMaxCurrent"` // optional
	FinishTime    string `mapstructure:"finishTime"`    // optional
}

type haServices struct {
	Start  string `mapstructure:"start_charging"` // script.*  optional
	Stop   string `mapstructure:"stop_charging"`  // script.*  optional
	Wakeup string `mapstructure:"wakeup"`         // script.*  optional
}

type haConfig struct {
	embed    `mapstructure:",squash"`
	Host     string     `mapstructure:"host"`  // http://ha:8123
	Token    string     `mapstructure:"token"` // Long-Lived Token
	Sensors  haSensors  `mapstructure:"sensors"`
	Services haServices `mapstructure:"services"`
}

// -----------------------------
// Adapter implementation
// -----------------------------

type HomeAssistant struct {
	*embed
	*request.Helper
	conf         haConfig
	startScript  string
	stopScript   string
	wakeupScript string
}

// Register on startup
func init() { registry.Add("homeassistant", newHAFromConfig) }

// Constructor from YAML config
func newHAFromConfig(other map[string]any) (api.Vehicle, error) {
	var cc haConfig
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	switch {
	case cc.Host == "":
		return nil, fmt.Errorf("homeassistant: host missing")
	case cc.Token == "":
		return nil, fmt.Errorf("homeassistant: token missing")
	case cc.Sensors.Soc == "":
		return nil, fmt.Errorf("homeassistant: sensors.soc missing")
	}

	log := util.NewLogger("ha-vehicle").Redact(cc.Token)
	return &HomeAssistant{
		embed:        &cc.embed,
		Helper:       request.NewHelper(log),
		conf:         cc,
		startScript:  cc.Services.Start,
		stopScript:   cc.Services.Stop,
		wakeupScript: cc.Services.Wakeup,
	}, nil
}

// -----------------------------
// Internal helper functions
// -----------------------------

// Calls /api/states/<entity> and returns .state
func (v *HomeAssistant) getState(entity string) (string, error) {
	uri := fmt.Sprintf("%s/api/states/%s", strings.TrimSuffix(v.conf.Host, "/"), entity)
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": "Bearer " + v.conf.Token,
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		State string `json:"state"`
	}
	if err = v.DoJSON(req, &resp); err != nil {
		return "", err
	}
	return resp.State, nil
}

// Calls script.<name> as a service call without payload
func (v *HomeAssistant) callScript(script string) error {
	if script == "" {
		return api.ErrNotAvailable
	}
	domain, name, ok := strings.Cut(script, ".")
	if !ok { // kein Punkt gefunden
		return fmt.Errorf("homeassistant: invalid script name '%s'", script)
	}
	uri := fmt.Sprintf("%s/api/services/%s/%s", v.conf.Host, domain, name)

	req, err := request.New(http.MethodPost, uri, bytes.NewBuffer([]byte("{}")), map[string]string{
		"Authorization": "Bearer " + v.conf.Token,
		"Content-Type":  "application/json",
	})
	if err != nil {
		return err
	}
	_, err = v.DoBody(req)
	return err
}

// -----------------------------
// Implementation of API interfaces
// -----------------------------

// generic helpers for fetching and parsing sensor values
func (v *HomeAssistant) getFloatSensor(entity string) (float64, error) {
	if entity == "" {
		return 0, api.ErrNotAvailable
	}
	s, err := v.getState(entity)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}

func (v *HomeAssistant) getIntSensor(entity string) (int64, error) {
	if entity == "" {
		return 0, api.ErrNotAvailable
	}
	s, err := v.getState(entity)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}

func (v *HomeAssistant) getBoolSensor(entity string) (bool, error) {
	if entity == "" {
		return false, api.ErrNotAvailable
	}
	s, err := v.getState(entity)
	if err != nil {
		return false, err
	}
	state := strings.ToLower(s)
	return state == "on" || state == "true" || state == "1" || state == "active", nil
}

func (v *HomeAssistant) getTimeSensor(entity string) (time.Time, error) {
	if entity == "" {
		return time.Time{}, api.ErrNotAvailable
	}
	s, err := v.getState(entity)
	if err != nil {
		return time.Time{}, err
	}
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(ts, 0), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	return t, err
}

// Soc returns state of charge [%]
func (v *HomeAssistant) Soc() (float64, error) {
	return v.getFloatSensor(v.conf.Sensors.Soc)
}

// Range returns remaining range [km] (optional)
func (v *HomeAssistant) Range() (int64, error) {
	return v.getIntSensor(v.conf.Sensors.Range)
}

// LimitSoc returns target state of charge [%] (optional)
func (v *HomeAssistant) LimitSoc() (float64, error) {
	return v.getFloatSensor(v.conf.Sensors.LimitSoc)
}

// Odometer returns odometer reading [km] (optional)
func (v *HomeAssistant) Odometer() (float64, error) {
	return v.getFloatSensor(v.conf.Sensors.Odometer)
}

// Climater returns true if climatization is active (optional)
func (v *HomeAssistant) Climater() (bool, error) {
	return v.getBoolSensor(v.conf.Sensors.Climater)
}

// MaxCurrent returns the maximum charging current (optional)
func (v *HomeAssistant) MaxCurrent() (int64, error) {
	return v.getIntSensor(v.conf.Sensors.MaxCurrent)
}

// GetMaxCurrent returns the currently set maximum charging current (optional)
func (v *HomeAssistant) GetMaxCurrent() (int64, error) {
	return v.getIntSensor(v.conf.Sensors.GetMaxCurrent)
}

// FinishTime returns the planned charging end time (optional)
func (v *HomeAssistant) FinishTime() (time.Time, error) {
	return v.getTimeSensor(v.conf.Sensors.FinishTime)
}

// GetLimitSoc implements the api.SocLimiter interface
func (v *HomeAssistant) GetLimitSoc() (int64, error) {
	return v.getIntSensor(v.conf.Sensors.LimitSoc)
}

// Status returns evcc charge status (optional)
func (v *HomeAssistant) Status() (api.ChargeStatus, error) {
	if v.conf.Sensors.Status == "" {
		return api.StatusNone, api.ErrNotAvailable
	}
	s, err := v.getState(v.conf.Sensors.Status)
	if err != nil {
		return api.StatusNone, err
	}

	switch strings.ToLower(s) {
	case "charging", "on", "true", "active":
		return api.StatusC, nil // Laden
	case "connected", "ready", "plugged":
		return api.StatusB, nil // Angesteckt
	default:
		return api.StatusA, nil // Getrennt
	}
}

// StartCharge triggers the start charging script
func (v *HomeAssistant) StartCharge() error { return v.callScript(v.startScript) }

// StopCharge triggers the stop charging script
func (v *HomeAssistant) StopCharge() error { return v.callScript(v.stopScript) }

// ChargeEnable toggles charging on/off (UI toggle)
func (v *HomeAssistant) ChargeEnable(enable bool) error {
	if enable {
		return v.StartCharge()
	}
	return v.StopCharge()
}

// WakeUp triggers the wakeup script (optional)
func (v *HomeAssistant) WakeUp() error {
	if v.wakeupScript == "" {
		return api.ErrNotAvailable
	}
	return v.callScript(v.wakeupScript)
}

// -----------------------------
// Compile-time checks
// -----------------------------

var (
	_ api.Vehicle            = (*HomeAssistant)(nil)
	_ api.VehicleRange       = (*HomeAssistant)(nil)
	_ api.ChargeState        = (*HomeAssistant)(nil)
	_ api.VehicleOdometer    = (*HomeAssistant)(nil)
	_ api.SocLimiter         = (*HomeAssistant)(nil)
	_ api.Resurrector        = (*HomeAssistant)(nil)
	_ api.VehicleClimater    = (*HomeAssistant)(nil)
	_ api.VehicleFinishTimer = (*HomeAssistant)(nil)
)

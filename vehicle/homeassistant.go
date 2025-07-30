package vehicle

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type haSensors struct {
	Soc           string // required
	Range         string // optional
	Status        string // optional
	LimitSoc      string // optional
	Odometer      string // optional
	Climater      string // optional
	MaxCurrent    string // optional
	GetMaxCurrent string // optional
	FinishTime    string // optional
}

type haServices struct {
	Start  string `mapstructure:"start_charging"` // script.*  optional
	Stop   string `mapstructure:"stop_charging"`  // script.*  optional
	Wakeup string // script.*  optional
}

type haConfig struct {
	embed    `mapstructure:",squash"`
	Host     string // http://ha:8123
	Token    string // Long-Lived Token
	Sensors  haSensors
	Services haServices
}

type HomeAssistant struct {
	*embed
	*request.Helper
	conf haConfig
	host string // host without trailing slash
	log  *util.Logger
}

// Register on startup
func init() {
	registry.Add("homeassistant", newHAFromConfig)
}

// Constructor from YAML config
func newHAFromConfig(other map[string]any) (api.Vehicle, error) {
	var cc haConfig
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	switch {
	case cc.Host == "":
		return nil, fmt.Errorf("missing host")
	case cc.Token == "":
		return nil, fmt.Errorf("missing token")
	case cc.Sensors.Soc == "":
		return nil, fmt.Errorf("missing soc sensor")
	}

	log := util.NewLogger("ha-vehicle").Redact(cc.Token)
	base := &HomeAssistant{
		embed:  &cc.embed,
		Helper: request.NewHelper(log),
		conf:   cc,
		host:   strings.TrimSuffix(cc.Host, "/"),
		log:    log,
	}

	// prepare optional feature functions
	var (
		vehicleRange    func() (int64, error)
		vehicleOdometer func() (float64, error)
		vehicleClimater func() (bool, error)
		vehicleFinish   func() (time.Time, error)
		maxCurrent      func(int64) error
		getMaxCurrent   func() (float64, error)
	)

	if cc.Sensors.Range != "" {
		vehicleRange = func() (int64, error) { return base.getIntSensor(cc.Sensors.Range) }
	}
	if cc.Sensors.Odometer != "" {
		vehicleOdometer = func() (float64, error) { return base.getFloatSensor(cc.Sensors.Odometer) }
	}
	if cc.Sensors.Climater != "" {
		vehicleClimater = func() (bool, error) { return base.getBoolSensor(cc.Sensors.Climater) }
	}
	if cc.Sensors.FinishTime != "" {
		vehicleFinish = func() (time.Time, error) { return base.getTimeSensor(cc.Sensors.FinishTime) }
	}
	if cc.Sensors.MaxCurrent != "" {
		// MaxCurrent is not a setter in HomeAssistant, so leave nil or implement if needed
	}
	if cc.Sensors.GetMaxCurrent != "" {
		getMaxCurrent = func() (float64, error) {
			v, err := base.getIntSensor(cc.Sensors.GetMaxCurrent)
			return float64(v), err
		}
	}

	// decorate all features
	return decorateVehicle(
		base,
		base.GetLimitSoc,
		base.Status,
		vehicleRange,
		vehicleOdometer,
		vehicleClimater,
		maxCurrent,
		getMaxCurrent,
		vehicleFinish,
		base.WakeUp,
		base.ChargeEnable,
	), nil
}

// Calls /api/states/<entity> and returns .state
func (v *HomeAssistant) getState(entity string) (string, error) {
	uri := fmt.Sprintf("%s/api/states/%s", v.host, url.PathEscape(entity))
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

/*
callScript calls script.<name> as a Home Assistant service call without payload.

Note: This function currently does not support passing parameters to scripts.
If you need to call scripts with parameters, payload support should be added in the future.

TODO: Add support for script parameters/payloads if needed.
*/
func (v *HomeAssistant) callScript(script string) error {
	domain, name, ok := strings.Cut(script, ".")
	if !ok { // kein Punkt gefunden
		return fmt.Errorf("invalid script name '%s'", script)
	}

	uri := fmt.Sprintf("%s/api/services/%s/%s", v.host, url.PathEscape(domain), url.PathEscape(name))

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

// generic helpers for fetching and parsing sensor values
func (v *HomeAssistant) getFloatSensor(entity string) (float64, error) {
	s, err := v.getState(entity)
	if err != nil {
		return 0, err
	}

	if s == "unknown" || s == "unavailable" {
		return 0, api.ErrNotAvailable
	}

	return strconv.ParseFloat(s, 64)
}

func (v *HomeAssistant) getIntSensor(entity string) (int64, error) {
	s, err := v.getState(entity)
	if err != nil {
		return 0, err
	}

	if s == "unknown" || s == "unavailable" {
		return 0, api.ErrNotAvailable
	}

	return strconv.ParseInt(s, 10, 64)
}

func (v *HomeAssistant) getBoolSensor(entity string) (bool, error) {
	s, err := v.getState(entity)
	if err != nil {
		return false, err
	}

	if s == "unknown" || s == "unavailable" {
		return false, api.ErrNotAvailable
	}

	state := strings.ToLower(s)
	return state == "on" || state == "true" || state == "1" || state == "active", nil
}

func (v *HomeAssistant) getTimeSensor(entity string) (time.Time, error) {
	s, err := v.getState(entity)
	if err != nil {
		return time.Time{}, err
	}

	if s == "unknown" || s == "unavailable" {
		return time.Time{}, api.ErrNotAvailable
	}

	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(ts, 0), nil
	}

	return time.Parse(time.RFC3339, s)
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

// SetMaxCurrent sets the maximum charging current via Home Assistant number.set_value service (if configured)
func (v *HomeAssistant) SetMaxCurrent(current int64) error {
	entity := v.conf.Sensors.MaxCurrent
	if entity == "" {
		return api.ErrNotAvailable
	}
	// Home Assistant expects a POST to /api/services/number/set_value with entity_id and value
	uri := fmt.Sprintf("%s/api/services/number/set_value", v.host)
	payload := fmt.Sprintf(`{"entity_id": "%s", "value": %d}`, entity, current)
	req, err := request.New(http.MethodPost, uri, bytes.NewBuffer([]byte(payload)), map[string]string{
		"Authorization": "Bearer " + v.conf.Token,
		"Content-Type":  "application/json",
	})
	if err != nil {
		return err
	}
	_, err = v.DoBody(req)
	return err
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
	var haStatusMap = map[string]api.ChargeStatus{
		"charging":            api.StatusC,
		"on":                  api.StatusC,
		"true":                api.StatusC,
		"active":              api.StatusC,
		"connected":           api.StatusB,
		"ready":               api.StatusB,
		"plugged":             api.StatusB,
		"disconnected":        api.StatusA,
		"off":                 api.StatusA,
		"none":                api.StatusA,
		"unavailable":         api.StatusA,
		"unknown":             api.StatusA,
		"notreadyforcharging": api.StatusA,
	}

	s, err := v.getState(v.conf.Sensors.Status)
	if err != nil {
		return api.StatusNone, err
	}

	state := strings.ToLower(s)
	if mapped, ok := haStatusMap[state]; ok {
		return mapped, nil
	}

	return api.StatusA, errors.New("invalid state: " + s)
}

// startCharge triggers the Home Assistant start charging script (if configured).
// Returns api.ErrNotAvailable if no start script is set.
func (v *HomeAssistant) startCharge() error {
	return v.callScript(v.conf.Services.Start)
}

// stopCharge triggers the Home Assistant stop charging script (if configured).
// Returns api.ErrNotAvailable if no stop script is set.
func (v *HomeAssistant) stopCharge() error {
	return v.callScript(v.conf.Services.Stop)
}

// ChargeEnable enables or disables charging via the UI toggle.
// Calls StartCharge or StopCharge accordingly.
func (v *HomeAssistant) ChargeEnable(enable bool) error {
	if enable {
		return v.startCharge()
	}
	return v.stopCharge()
}

// WakeUp triggers the Home Assistant wakeup script (if configured).
// Returns api.ErrNotAvailable if no wakeup script is set.
func (v *HomeAssistant) WakeUp() error {
	if v.conf.Services.Wakeup == "" {
		return api.ErrNotAvailable
	}
	return v.callScript(v.conf.Services.Wakeup)
}

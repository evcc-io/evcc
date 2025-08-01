package vehicle

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Public wrappers to satisfy api.Vehicle interface, call function variables if set
func (v *HomeAssistant) Soc() (float64, error) {
	if v.socFunc != nil {
		return v.socFunc()
	}
	return 0, api.ErrNotAvailable
}
func (v *HomeAssistant) Range() (int64, error) {
	if v.rangeFunc != nil {
		return v.rangeFunc()
	}
	return 0, api.ErrNotAvailable
}
func (v *HomeAssistant) Odometer() (float64, error) {
	if v.odoFunc != nil {
		return v.odoFunc()
	}
	return 0, api.ErrNotAvailable
}
func (v *HomeAssistant) Climater() (bool, error) {
	if v.climaterFunc != nil {
		return v.climaterFunc()
	}
	return false, api.ErrNotAvailable
}
func (v *HomeAssistant) FinishTime() (time.Time, error) {
	if v.finishFunc != nil {
		return v.finishFunc()
	}
	return time.Time{}, api.ErrNotAvailable
}
func (v *HomeAssistant) LimitSoc() (float64, error) {
	if v.limitSocFunc != nil {
		return v.limitSocFunc()
	}
	return 0, api.ErrNotAvailable
}

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
	URI      string // http://homeassistant:8123
	Token    string // Long-Lived Token
	Sensors  haSensors
	Services haServices
}

type HomeAssistant struct {
	*embed
	*request.Helper
	conf haConfig
	uri  string

	socFunc      func() (float64, error)
	rangeFunc    func() (int64, error)
	odoFunc      func() (float64, error)
	climaterFunc func() (bool, error)
	finishFunc   func() (time.Time, error)
	limitSocFunc func() (float64, error)
}

// Status implements api.ChargeState (optional)
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

	return api.StatusA, fmt.Errorf("invalid state: %s", s)
}

// ChargeEnable implements api.ChargeController (start/stop charging via Home Assistant script)
func (v *HomeAssistant) ChargeEnable(enable bool) error {
	if enable {
		return v.callScript(v.conf.Services.Start)
	}
	return v.callScript(v.conf.Services.Stop)
}

// WakeUp implements api.Resurrector (optional)
func (v *HomeAssistant) WakeUp() error {
	if v.conf.Services.Wakeup == "" {
		return api.ErrNotAvailable
	}
	return v.callScript(v.conf.Services.Wakeup)
}

// Register on startup
func init() {
	registry.Add("homeassistant", newHomeAssistantVehicleFromConfig)
}

// Constructor from YAML config
func newHomeAssistantVehicleFromConfig(other map[string]any) (api.Vehicle, error) {
	var cc haConfig
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	switch {
	case cc.URI == "":
		return nil, fmt.Errorf("missing uri")
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
		uri:    strings.TrimSuffix(cc.URI, "/"),
	}
	// Set up custom transport to always add Authorization and Content-Type headers
	base.Helper.Client.Transport = &transport.Decorator{
		Decorator: transport.DecorateHeaders(map[string]string{
			"Authorization": "Bearer " + cc.Token,
			"Content-Type":  "application/json",
		}),
		Base: http.DefaultTransport,
	}

	// assign all feature funcs to struct fields for clarity, only if configured
	if cc.Sensors.Soc != "" {
		base.socFunc = base.soc
	}
	if cc.Sensors.Range != "" {
		base.rangeFunc = base.rangeKm
	}
	if cc.Sensors.Odometer != "" {
		base.odoFunc = base.odometer
	}
	if cc.Sensors.Climater != "" {
		base.climaterFunc = base.climater
	}
	if cc.Sensors.FinishTime != "" {
		base.finishFunc = base.finishTime
	}
	if cc.Sensors.LimitSoc != "" {
		base.limitSocFunc = base.limitSoc
	}

	return base, nil
}

// Calls /api/states/<entity> and returns .state
func (v *HomeAssistant) getState(entity string) (string, error) {
	uri := fmt.Sprintf("%s/api/states/%s", v.uri, url.PathEscape(entity))
	req, err := request.New(http.MethodGet, uri, nil, nil)
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

func (v *HomeAssistant) callScript(script string) error {
	domain, name, ok := strings.Cut(script, ".")
	if !ok { // kein Punkt gefunden
		return fmt.Errorf("invalid script name '%s'", script)
	}

	uri := fmt.Sprintf("%s/api/services/%s/%s", v.uri, url.PathEscape(domain), url.PathEscape(name))

	req, err := request.New(http.MethodPost, uri, bytes.NewBuffer([]byte("{}")), map[string]string{
		"Content-Type": "application/json",
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

// soc returns state of charge [%] (private)
func (v *HomeAssistant) soc() (float64, error) {
	return v.getFloatSensor(v.conf.Sensors.Soc)
}

// rangeKm returns remaining range [km] (private)
func (v *HomeAssistant) rangeKm() (int64, error) {
	return v.getIntSensor(v.conf.Sensors.Range)
}

// limitSoc returns target state of charge [%] (private)
func (v *HomeAssistant) limitSoc() (float64, error) {
	return v.getFloatSensor(v.conf.Sensors.LimitSoc)
}

// odometer returns odometer reading [km] (private)
func (v *HomeAssistant) odometer() (float64, error) {
	return v.getFloatSensor(v.conf.Sensors.Odometer)
}

// climater returns true if climatization is active (private)
func (v *HomeAssistant) climater() (bool, error) {
	return v.getBoolSensor(v.conf.Sensors.Climater)
}

// finishTime returns the planned charging end time (private)
func (v *HomeAssistant) finishTime() (time.Time, error) {
	return v.getTimeSensor(v.conf.Sensors.FinishTime)
}

// GetLimitSoc implements api.SocLimiter (Decorator-Interface)
func (v *HomeAssistant) GetLimitSoc() (int64, error) {
	if v.limitSocFunc != nil {
		val, err := v.limitSocFunc()
		return int64(val), err
	}
	return 0, api.ErrNotAvailable
}

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
	"github.com/evcc-io/evcc/util/transport"
)

type HomeAssistant struct {
	*embed
	*request.Helper
	uri string
	soc string
}

// Register on startup
func init() {
	registry.Add("homeassistant", NewHomeAssistantVehicleFromConfig)
}

// Constructor from YAML config
func NewHomeAssistantVehicleFromConfig(other map[string]any) (api.Vehicle, error) {
	var cc struct {
		embed   `mapstructure:",squash"`
		URI     string
		Token   string
		Sensors struct {
			Soc        string // required
			Range      string // optional
			Status     string // optional
			LimitSoc   string // optional
			Odometer   string // optional
			Climater   string // optional
			FinishTime string // optional
		}
		Services struct {
			Start  string `mapstructure:"start_charging"` // script.*  optional
			Stop   string `mapstructure:"stop_charging"`  // script.*  optional
			Wakeup string // script.*  optional
		}
	}

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

	res := &HomeAssistant{
		embed:  &cc.embed,
		Helper: request.NewHelper(util.NewLogger("ha-vehicle").Redact(cc.Token)),
		uri:    strings.TrimSuffix(cc.URI, "/"),
		soc:    cc.Sensors.Soc,
	}

	res.Client.Transport = &transport.Decorator{
		Base: res.Client.Transport,
		Decorator: transport.DecorateHeaders(map[string]string{
			"Authorization": "Bearer " + cc.Token,
		}),
	}

	// prepare optional feature functions with concise names
	var (
		limitSoc     func() (int64, error)
		status       func() (api.ChargeStatus, error)
		rng          func() (int64, error)
		odo          func() (float64, error)
		climater     func() (bool, error)
		finish       func() (time.Time, error)
		chargeEnable func(bool) error
		wakeup       func() error
	)

	if cc.Sensors.Status != "" {
		status = func() (api.ChargeStatus, error) { return res.status(cc.Sensors.Status) }
	}
	if cc.Sensors.Range != "" {
		rng = func() (int64, error) { return res.getIntSensor(cc.Sensors.Range) }
	}
	if cc.Sensors.Odometer != "" {
		odo = func() (float64, error) { return res.getFloatSensor(cc.Sensors.Odometer) }
	}
	if cc.Sensors.Climater != "" {
		climater = func() (bool, error) { return res.getBoolSensor(cc.Sensors.Climater) }
	}
	if cc.Sensors.FinishTime != "" {
		finish = func() (time.Time, error) { return res.getTimeSensor(cc.Sensors.FinishTime) }
	}
	if cc.Services.Start != "" && cc.Services.Stop != "" {
		chargeEnable = func(enable bool) error {
			if enable {
				return res.callScript(cc.Services.Start)
			}
			return res.callScript(cc.Services.Stop)
		}
	}
	if cc.Services.Wakeup != "" {
		wakeup = func() error {
			return res.callScript(cc.Services.Wakeup)
		}
	}

	// decorate all features
	return decorateVehicle(
		res,
		limitSoc,
		status,
		rng,
		odo,
		climater,
		nil, // maxCurrent setter not implemented
		nil, // getMaxCurrent getter not implemented
		finish,
		wakeup,
		chargeEnable,
	), nil
}

func (v *HomeAssistant) Soc() (float64, error) {
	return v.getFloatSensor(v.soc)
}

// Calls /api/states/<entity> and returns .state
func (v *HomeAssistant) getState(entity string) (string, error) {
	var res struct {
		State string `json:"state"`
	}

	uri := fmt.Sprintf("%s/api/states/%s", v.uri, url.PathEscape(entity))
	err := v.GetJSON(uri, &res)

	return res.State, err
}

func (v *HomeAssistant) callScript(script string) error {
	domain, name, ok := strings.Cut(script, ".")
	if !ok { // kein Punkt gefunden
		return fmt.Errorf("invalid script name '%s'", script)
	}

	uri := fmt.Sprintf("%s/api/services/%s/%s", v.uri, url.PathEscape(domain), url.PathEscape(name))
	req, _ := request.New(http.MethodPost, uri, bytes.NewBuffer([]byte("{}")))

	_, err := v.DoBody(req)
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

// status returns evcc charge status (optional, private)
func (v *HomeAssistant) status(sensor string) (api.ChargeStatus, error) {
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

	s, err := v.getState(sensor)
	if err != nil {
		return api.StatusNone, err
	}

	state := strings.ToLower(s)
	if mapped, ok := haStatusMap[state]; ok {
		return mapped, nil
	}

	return api.StatusNone, errors.New("invalid state: " + s)
}

package vehicle

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
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
		embed           `mapstructure:",squash"`
		URI             string
		Token           string
		Soc             string // required
		Range           string // optional
		Status          string // optional
		LimitSoc        string // optional
		Odometer        string // optional
		Climater        string // optional
		FinishTime      string // optional
		GetMaxCurrent   string `mapstructure:"getMaxCurrent"` // optional
		StartCharging   string `mapstructure:"start_charging"` // script.*  optional
		StopCharging    string `mapstructure:"stop_charging"`  // script.*  optional
		Wakeup          string // script.*  optional
		SetMaxCurrent   string `mapstructure:"setMaxCurrent"` // script.*  optional
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	switch {
	case cc.URI == "":
		return nil, errors.New("missing uri")
	case cc.Token == "":
		return nil, errors.New("missing token")
	case cc.Soc == "":
		return nil, errors.New("missing soc sensor")
	}

	res := &HomeAssistant{
		embed:  &cc.embed,
		Helper: request.NewHelper(util.NewLogger("ha-vehicle").Redact(cc.Token)),
		uri:    strings.TrimSuffix(cc.URI, "/"),
		soc:    cc.Soc,
	}

	res.Client.Transport = &transport.Decorator{
		Base: res.Client.Transport,
		Decorator: transport.DecorateHeaders(map[string]string{
			"Authorization": "Bearer " + cc.Token,
		}),
	}

	// prepare optional feature functions with concise names
	var (
		limitSoc      func() (int64, error)
		status        func() (api.ChargeStatus, error)
		rng           func() (int64, error)
		odo           func() (float64, error)
		climater      func() (bool, error)
		finish        func() (time.Time, error)
		chargeEnable  func(bool) error
		wakeup        func() error
		setMaxCurrent func(int64) error
		getMaxCurrent func() (float64, error)
	)

	if cc.LimitSoc != "" {
		limitSoc = func() (int64, error) { return res.getIntSensor(cc.LimitSoc) }
	}
	if cc.Status != "" {
		status = func() (api.ChargeStatus, error) { return res.status(cc.Status) }
	}
	if cc.Range != "" {
		rng = func() (int64, error) { return res.getIntSensor(cc.Range) }
	}
	if cc.Odometer != "" {
		odo = func() (float64, error) { return res.getFloatSensor(cc.Odometer) }
	}
	if cc.Climater != "" {
		climater = func() (bool, error) { return res.getBoolSensor(cc.Climater) }
	}
	if cc.FinishTime != "" {
		finish = func() (time.Time, error) { return res.getTimeSensor(cc.FinishTime) }
	}
	if cc.GetMaxCurrent != "" {
		getMaxCurrent = func() (float64, error) { return res.getFloatSensor(cc.GetMaxCurrent) }
	}
	if cc.StartCharging != "" && cc.StopCharging != "" {
		chargeEnable = func(enable bool) error {
			if enable {
				return res.callScript(cc.StartCharging)
			}
			return res.callScript(cc.StopCharging)
		}
	}
	if cc.Wakeup != "" {
		wakeup = func() error { return res.callScript(cc.Wakeup) }
	}
	if cc.SetMaxCurrent != "" {
		setMaxCurrent = func(current int64) error {
			return res.callScriptWithCurrent(cc.SetMaxCurrent, current)
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
		setMaxCurrent,
		getMaxCurrent,
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
	if err := v.GetJSON(uri, &res); err != nil {
		return "", err
	}

	if res.State == "unknown" || res.State == "unavailable" {
		return "", api.ErrNotAvailable
	}

	return res.State, nil
}

func (v *HomeAssistant) callScript(script string) error {
	// All configured services are scripts, so always use script.turn_on
	payload := fmt.Sprintf(`{"entity_id": "%s"}`, script)
	uri := fmt.Sprintf("%s/api/services/script/turn_on", v.uri)
	req, _ := request.New(http.MethodPost, uri, strings.NewReader(payload))
	_, err := v.DoBody(req)
	return err
}

func (v *HomeAssistant) callScriptWithCurrent(script string, current int64) error {
	// Call script with current parameter
	payload := fmt.Sprintf(`{"entity_id": "%s", "variables": {"current": %d}}`, script, current)
	uri := fmt.Sprintf("%s/api/services/script/turn_on", v.uri)
	req, _ := request.New(http.MethodPost, uri, strings.NewReader(payload))
	_, err := v.DoBody(req)
	return err
}

// generic helpers for fetching and parsing sensor values
func (v *HomeAssistant) getFloatSensor(entity string) (float64, error) {
	s, err := v.getState(entity)
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(s, 64)
}

func (v *HomeAssistant) getIntSensor(entity string) (int64, error) {
	s, err := v.getState(entity)
	if err != nil {
		return 0, err
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(f), nil // truncation
}

func (v *HomeAssistant) getBoolSensor(entity string) (bool, error) {
	s, err := v.getState(entity)
	if err != nil {
		return false, err
	}

	return slices.Contains([]string{"on", "true", "1", "active"}, strings.ToLower(s)), nil
}

func (v *HomeAssistant) getTimeSensor(entity string) (time.Time, error) {
	s, err := v.getState(entity)
	if err != nil {
		return time.Time{}, err
	}

	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(ts, 0), nil
	}

	return time.Parse(time.RFC3339, s)
}

// status returns evcc charge status (optional, private)
func (v *HomeAssistant) status(sensor string) (api.ChargeStatus, error) {
	var haStatusMap = map[string]api.ChargeStatus{
		"c":                   api.StatusC,
		"charging":            api.StatusC,
		"on":                  api.StatusC,
		"true":                api.StatusC,
		"active":              api.StatusC,
		"b":                   api.StatusB,
		"connected":           api.StatusB,
		"ready":               api.StatusB,
		"plugged":             api.StatusB,
		"charging_completed":  api.StatusB,
		"initialising":        api.StatusB,
		"a":                   api.StatusA,
		"disconnected":        api.StatusA,
		"off":                 api.StatusA,
		"none":                api.StatusA,
		"unavailable":         api.StatusA,
		"unknown":             api.StatusA,
		"notreadyforcharging": api.StatusA,
		"not_plugged":         api.StatusA,
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

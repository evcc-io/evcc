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
	log *util.Logger
	uri string
	soc string
}

// Register on startup
func init() {
	registry.Add("homeassistant", NewHomeAssistantVehicleFromConfig)
}

// Constructor from YAML config
func NewHomeAssistantVehicleFromConfig(other map[string]any) (api.Vehicle, error) {
	// Remove conflicting top-level parameters that would interfere with OnIdentify mapping
	maxCurrentEntity := ""
	getMaxCurrentEntity := ""
	
	if val, ok := other["maxCurrent"]; ok {
		if str, ok := val.(string); ok {
			maxCurrentEntity = str
		}
		delete(other, "maxCurrent") // Remove to prevent OnIdentify.MaxCurrent conflict
	}
	
	if val, ok := other["getMaxCurrent"]; ok {
		if str, ok := val.(string); ok {
			getMaxCurrentEntity = str
		}
		delete(other, "getMaxCurrent") // Remove to prevent OnIdentify conflict
	}

	var cc struct {
		embed   `mapstructure:",squash"`
		URI     string
		Token   string
		Sensors struct {
			Soc           string // required
			Range         string // optional
			Status        string // optional
			LimitSoc      string // optional
			Odometer      string // optional
			Climater      string // optional
			FinishTime    string // optional
			MaxCurrent    string // optional - number.* or input_number.* entity for max charge current setter
			GetMaxCurrent string `mapstructure:"getMaxCurrent"` // optional - sensor.*, number.* or input_number.* entity for max charge current getter
		} `mapstructure:"sensors"`
		Services struct {
			Start  string `mapstructure:"start_charging"` // script.*  optional
			Stop   string `mapstructure:"stop_charging"`  // script.*  optional
			Wakeup string // script.*  optional
		}
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// Set the extracted sensor entity values
	if maxCurrentEntity != "" {
		cc.Sensors.MaxCurrent = maxCurrentEntity
	}
	if getMaxCurrentEntity != "" {
		cc.Sensors.GetMaxCurrent = getMaxCurrentEntity
	}

	switch {
	case cc.URI == "":
		return nil, errors.New("missing uri")
	case cc.Token == "":
		return nil, errors.New("missing token")
	case cc.Sensors.Soc == "":
		return nil, errors.New("missing soc sensor")
	}

	log := util.NewLogger("ha-vehicle").Redact(cc.Token)
	res := &HomeAssistant{
		embed:  &cc.embed,
		Helper: request.NewHelper(log),
		log:    log,
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

		// new: max current getter/setter
		setMaxCurrent func(int64) error
		getMaxCurrent func() (float64, error)

		wakeup func() error
	)

	if cc.Sensors.LimitSoc != "" {
		limitSoc = func() (int64, error) { return res.getIntSensor(cc.Sensors.LimitSoc) }
	}
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
		wakeup = func() error { return res.callScript(cc.Services.Wakeup) }
	}

	// implement maxCurrent getter/setter if configured (number.* or input_number.* entity expected)
	// NOTE: The MaxCurrent setter will only be called if the vehicle is used as a charger (not by default from the loadpoint logic).
	// For Home Assistant, you must explicitly call the vehicle's MaxCurrent if you want to propagate changes from the loadpoint.
	if cc.Sensors.MaxCurrent != "" {
		setMaxCurrent = func(value int64) error {
			// determine domain (number.* or input_number.*)
			parts := strings.SplitN(cc.Sensors.MaxCurrent, ".", 2)
			if len(parts) != 2 {
				return errors.New("invalid entity id")
			}
			domain := parts[0]
			var svc string
			switch domain {
			case "number":
				svc = "number/set_value"
			case "input_number":
				svc = "input_number/set_value"
			default:
				return errors.New("unsupported entity domain for maxCurrent: " + domain)
			}

			payload := fmt.Sprintf(`{"entity_id": "%s", "value": %d}`, cc.Sensors.MaxCurrent, value)
			uri := fmt.Sprintf("%s/api/services/%s", res.uri, svc)
			req, _ := request.New(http.MethodPost, uri, strings.NewReader(payload), request.JSONEncoding)
			res.log.ERROR.Printf("TESTLOG: setMaxCurrent CALLED with value=%d uri=%s payload=%s", value, uri, payload)
			body, err := res.DoBody(req)
			if err != nil {
				res.log.ERROR.Printf("ha-vehicle: setMaxCurrent failed: %v uri=%s payload=%s response=%s", err, uri, payload, string(body))
			} else {
				res.log.DEBUG.Printf("ha-vehicle: setMaxCurrent response: %s", string(body))
			}
			return err
		}
	}

	// implement getMaxCurrent getter if configured separately (for reading current from HA)
	// supports sensor.*, number.*, or input_number.* entities
	if cc.Sensors.GetMaxCurrent != "" {
		getMaxCurrent = func() (float64, error) { return res.getFloatSensor(cc.Sensors.GetMaxCurrent) }
	} else if cc.Sensors.MaxCurrent != "" {
		// fallback: use MaxCurrent entity for both reading and writing
		getMaxCurrent = func() (float64, error) { return res.getFloatSensor(cc.Sensors.MaxCurrent) }
	}

	// decorate all features
	return decorateVehicle(
		res,
		limitSoc,
		status,
		rng,
		odo,
		climater,
		setMaxCurrent, // maxCurrent setter
		getMaxCurrent, // getMaxCurrent getter
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

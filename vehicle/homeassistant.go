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

// tokenTransport decorates all requests with the Authorization header
type tokenTransport struct {
	token     string
	transport http.RoundTripper
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+t.token)
	return t.transport.RoundTrip(req2)
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
	// Set up custom transport to always add Authorization header
	base.Helper.Client.Transport = &tokenTransport{
		token:     cc.Token,
		transport: http.DefaultTransport,
	}

	// prepare optional feature functions with concise names
	var (
		rng      func() (int64, error)
		odo      func() (float64, error)
		climater func() (bool, error)
		finish   func() (time.Time, error)
	)

	if cc.Sensors.Range != "" {
		rng = func() (int64, error) { return base.getIntSensor(cc.Sensors.Range) }
	}
	if cc.Sensors.Odometer != "" {
		odo = func() (float64, error) { return base.getFloatSensor(cc.Sensors.Odometer) }
	}
	if cc.Sensors.Climater != "" {
		climater = func() (bool, error) { return base.getBoolSensor(cc.Sensors.Climater) }
	}
	if cc.Sensors.FinishTime != "" {
		finish = func() (time.Time, error) { return base.getTimeSensor(cc.Sensors.FinishTime) }
	}

	// decorate all features
	return decorateVehicle(
		base,
		base.getLimitSoc,
		base.status,
		rng,
		odo,
		climater,
		nil, // maxCurrent setter not implemented
		nil, // getMaxCurrent getter not implemented
		finish,
		base.WakeUp,
		base.ChargeEnable,
	), nil
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

// FinishTime returns the planned charging end time (optional)
func (v *HomeAssistant) FinishTime() (time.Time, error) {
	return v.getTimeSensor(v.conf.Sensors.FinishTime)
}

// getLimitSoc implements the api.SocLimiter interface (private)
// Use float64 for consistency with the public LimitSoc method
// getLimitSoc implements the api.SocLimiter interface (private)
// Gibt int64 zurück, wie vom Decorator erwartet
func (v *HomeAssistant) getLimitSoc() (int64, error) {
	val, err := v.getFloatSensor(v.conf.Sensors.LimitSoc)
	if err != nil {
		return 0, err
	}
	return int64(val), nil
}

// status returns evcc charge status (optional, private)
func (v *HomeAssistant) status() (api.ChargeStatus, error) {
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
func (v *HomeAssistant) startCharge() error {
	return v.callScript(v.conf.Services.Start)
}

// stopCharge triggers the Home Assistant stop charging script (if configured).
func (v *HomeAssistant) stopCharge() error {
	return v.callScript(v.conf.Services.Stop)
}

// Calls StartCharge or StopCharge accordingly.
func (v *HomeAssistant) ChargeEnable(enable bool) error {
	if enable {
		return v.startCharge()
	}
	return v.stopCharge()
}

// WakeUp triggers the Home Assistant wakeup script (if configured).
func (v *HomeAssistant) WakeUp() error {
	if v.conf.Services.Wakeup == "" {
		return api.ErrNotAvailable
	}
	return v.callScript(v.conf.Services.Wakeup)
}

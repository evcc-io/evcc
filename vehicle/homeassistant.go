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
)

type haConfig struct {
	Soc        string `mapstructure:"soc"`        // required
	LimitSoc   string `mapstructure:"limitSoc"`   // optional
	Range      string `mapstructure:"range"`      // optional
	Odometer   string `mapstructure:"odometer"`   // optional
	Climater   string `mapstructure:"climater"`   // optional
	Status     string `mapstructure:"status"`     // optional
	Enable     string `mapstructure:"enable"`     // optional
	Wakeup     string `mapstructure:"wakeup"`     // optional
	FinishTime string `mapstructure:"finishTime"` // optional
}

type HomeAssistant struct {
	*embed
	*request.Helper
	conf          haConfig
	baseURL       string
	token         string
	startService  string
	stopService   string
	wakeupService string
}

func init() {
	registry.Add("homeassistant", newHomeAssistantFromConfig)
}

func newHomeAssistantFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed      `mapstructure:",squash"`
		URI        string `mapstructure:"uri"`
		Token      string `mapstructure:"token"`
		Soc        string `mapstructure:"soc"`
		LimitSoc   string `mapstructure:"limitSoc"`
		Range      string `mapstructure:"range"`
		Odometer   string `mapstructure:"odometer"`
		Climater   string `mapstructure:"climater"`
		Status     string `mapstructure:"status"`
		Enable     string `mapstructure:"enable"`
		Wakeup     string `mapstructure:"wakeup"`
		FinishTime string `mapstructure:"finishTime"`
		Services   struct {
			StartCharging string `mapstructure:"start_charging"`
			StopCharging  string `mapstructure:"stop_charging"`
			Wakeup        string `mapstructure:"wakeup"`
		} `mapstructure:"services"`
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("ha")

	// Use services if defined, otherwise fall back to enable field
	enableService := cc.Enable
	wakeupService := cc.Wakeup
	if cc.Services.StartCharging != "" && cc.Services.StopCharging != "" {
		enableService = "services" // mark that we have services configured
	}
	if cc.Services.Wakeup != "" {
		wakeupService = cc.Services.Wakeup
	}

	v := &HomeAssistant{
		embed:  &cc.embed,
		Helper: request.NewHelper(log),
		conf: haConfig{
			Soc:        cc.Soc,
			LimitSoc:   cc.LimitSoc,
			Range:      cc.Range,
			Odometer:   cc.Odometer,
			Climater:   cc.Climater,
			Status:     cc.Status,
			Enable:     enableService,
			Wakeup:     wakeupService,
			FinishTime: cc.FinishTime,
		},
		baseURL:       strings.TrimSuffix(cc.URI, "/"),
		token:         cc.Token,
		startService:  cc.Services.StartCharging,
		stopService:   cc.Services.StopCharging,
		wakeupService: cc.Services.Wakeup,
	}

	// For now, return the base vehicle without decoration
	// TODO: implement decorator pattern after go generate works
	return v, nil
}

// Soc implements api.Vehicle (mandatory)
func (v *HomeAssistant) Soc() (float64, error) {
	return v.getFloatSensor(v.conf.Soc)
}

// Optional interface implementations (conditional based on configuration)

// GetLimitSoc implements api.SocLimiter
func (v *HomeAssistant) GetLimitSoc() (int64, error) {
	if v.conf.LimitSoc == "" {
		return 0, api.ErrNotAvailable
	}
	return v.limitSoc()
}

// Range implements api.VehicleRange
func (v *HomeAssistant) Range() (int64, error) {
	if v.conf.Range == "" {
		return 0, api.ErrNotAvailable
	}
	return v.rangeKm()
}

// Odometer implements api.VehicleOdometer
func (v *HomeAssistant) Odometer() (float64, error) {
	if v.conf.Odometer == "" {
		return 0, api.ErrNotAvailable
	}
	return v.odometer()
}

// Climater implements api.VehicleClimater
func (v *HomeAssistant) Climater() (bool, error) {
	if v.conf.Climater == "" {
		return false, api.ErrNotAvailable
	}
	return v.climater()
}

// Status implements api.ChargeState
func (v *HomeAssistant) Status() (api.ChargeStatus, error) {
	if v.conf.Status == "" {
		return api.StatusNone, api.ErrNotAvailable
	}
	return v.status()
}

// ChargeEnable implements api.ChargeController
func (v *HomeAssistant) ChargeEnable(enable bool) error {
	if v.conf.Enable == "" {
		return api.ErrNotAvailable
	}
	return v.chargeEnable(enable)
}

// WakeUp implements api.Resurrector
func (v *HomeAssistant) WakeUp() error {
	if v.wakeupService == "" && v.conf.Wakeup == "" {
		return api.ErrNotAvailable
	}
	return v.wakeUp()
}

// FinishTime implements api.VehicleFinishTimer
func (v *HomeAssistant) FinishTime() (time.Time, error) {
	if v.conf.FinishTime == "" {
		return time.Time{}, api.ErrNotAvailable
	}
	return v.finishTime()
}

// private helper methods for decorator pattern

func (v *HomeAssistant) limitSoc() (int64, error) {
	val, err := v.getFloatSensor(v.conf.LimitSoc)
	if err != nil {
		return 0, err
	}
	return int64(val), nil
}

func (v *HomeAssistant) rangeKm() (int64, error) {
	s, err := v.getState(v.conf.Range)
	if err != nil {
		return 0, err
	}
	if s == "unknown" || s == "unavailable" {
		return 0, api.ErrNotAvailable
	}
	return strconv.ParseInt(s, 10, 64)
}

func (v *HomeAssistant) odometer() (float64, error) {
	return v.getFloatSensor(v.conf.Odometer)
}

func (v *HomeAssistant) climater() (bool, error) {
	s, err := v.getState(v.conf.Climater)
	if err != nil {
		return false, err
	}
	if s == "unknown" || s == "unavailable" {
		return false, api.ErrNotAvailable
	}
	state := strings.ToLower(s)
	return state == "on" || state == "true" || state == "1" || state == "active", nil
}

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

	s, err := v.getState(v.conf.Status)
	if err != nil {
		return api.StatusNone, err
	}

	state := strings.ToLower(s)
	if mapped, ok := haStatusMap[state]; ok {
		return mapped, nil
	}

	return api.StatusA, fmt.Errorf("invalid state: %s", s)
}

func (v *HomeAssistant) chargeEnable(enable bool) error {
	var service string

	// Use services if configured, otherwise fall back to enable field or defaults
	if v.startService != "" && v.stopService != "" {
		if enable {
			service = v.startService
		} else {
			service = v.stopService
		}
	} else if v.conf.Enable != "" {
		service = v.conf.Enable
	} else {
		// Default fallback
		if enable {
			service = "script.start_charging"
		} else {
			service = "script.stop_charging"
		}
	}

	return v.callScript(service)
}

func (v *HomeAssistant) wakeUp() error {
	// Use services wakeup if configured, otherwise fall back to top-level wakeup
	if v.wakeupService != "" {
		return v.callScript(v.wakeupService)
	} else if v.conf.Wakeup != "" {
		return v.callScript(v.conf.Wakeup)
	}
	return api.ErrNotAvailable
}

func (v *HomeAssistant) finishTime() (time.Time, error) {
	s, err := v.getState(v.conf.FinishTime)
	if err != nil {
		return time.Time{}, err
	}

	if s == "unknown" || s == "unavailable" || s == "" {
		return time.Time{}, api.ErrNotAvailable
	}

	// Try parsing as Unix timestamp first
	if unix, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(unix, 0), nil
	}

	// Try parsing as ISO8601/RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try parsing without timezone
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse finish time: %s", s)
}

// helper methods

func (v *HomeAssistant) getState(entity string) (string, error) {
	uri := fmt.Sprintf("%s/api/states/%s", v.baseURL, url.PathEscape(entity))

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": "Bearer " + v.token,
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		State string `json:"state"`
	}
	err = v.DoJSON(req, &resp)
	if err != nil {
		return "", err
	}

	return resp.State, nil
}

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

func (v *HomeAssistant) callScript(script string) error {
	domain, name, ok := strings.Cut(script, ".")
	if !ok {
		return fmt.Errorf("invalid script name '%s'", script)
	}


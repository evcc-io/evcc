package vehicle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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
	registry.Add("homeassistant", NewHomeAssistantFromConfig)
}

// NewHomeAssistantFromConfig creates a new HomeAssistant vehicle
func NewHomeAssistantFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	if cc.URI == "" || cc.Token == "" || cc.Soc == "" {
		return nil, fmt.Errorf("missing required configuration: uri, token, soc")
	}

	log := util.NewLogger("homeassistant")

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
			FinishTime: cc.FinishTime,
		},
		baseURL:       strings.TrimSuffix(cc.URI, "/"),
		token:         cc.Token,
		startService:  cc.Services.StartCharging,
		stopService:   cc.Services.StopCharging,
		wakeupService: cc.Services.Wakeup,
	}

	return v, nil
}

// Soc implements api.Vehicle (mandatory)
func (v *HomeAssistant) Soc() (float64, error) {
	return v.getFloatSensor(v.conf.Soc)
}

// GetLimitSoc implements api.SocLimiter (only if configured)
func (v *HomeAssistant) GetLimitSoc() (int64, error) {
	if v.conf.LimitSoc == "" {
		return 0, api.ErrNotAvailable
	}

	val, err := v.getFloatSensor(v.conf.LimitSoc)
	if err != nil {
		return 0, err
	}
	return int64(val), nil
}

// Status implements api.ChargeState (only if configured)
func (v *HomeAssistant) Status() (api.ChargeStatus, error) {
	if v.conf.Status == "" {
		return api.StatusNone, api.ErrNotAvailable
	}

	state, err := v.getState(v.conf.Status)
	if err != nil {
		return api.StatusNone, err
	}

	// Map Home Assistant charging states to EVCC states
	switch strings.ToLower(state) {
	case "charging":
		return api.StatusC, nil
	case "connected", "plugged":
		return api.StatusB, nil
	case "disconnected", "unplugged":
		return api.StatusA, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown status: %s", state)
	}
}

// Range implements api.VehicleRange (only if configured)
func (v *HomeAssistant) Range() (int64, error) {
	if v.conf.Range == "" {
		return 0, api.ErrNotAvailable
	}

	val, err := v.getFloatSensor(v.conf.Range)
	if err != nil {
		return 0, err
	}
	return int64(val), nil
}

// Odometer implements api.VehicleOdometer (only if configured)
func (v *HomeAssistant) Odometer() (float64, error) {
	if v.conf.Odometer == "" {
		return 0, api.ErrNotAvailable
	}

	return v.getFloatSensor(v.conf.Odometer)
}

// Climater implements api.VehicleClimater (only if configured)
func (v *HomeAssistant) Climater() (bool, error) {
	if v.conf.Climater == "" {
		return false, api.ErrNotAvailable
	}

	state, err := v.getState(v.conf.Climater)
	if err != nil {
		return false, err
	}
	return strings.ToLower(state) == "on", nil
}

// FinishTime implements api.VehicleFinishTimer (only if configured)
func (v *HomeAssistant) FinishTime() (time.Time, error) {
	if v.conf.FinishTime == "" {
		return time.Time{}, api.ErrNotAvailable
	}

	state, err := v.getState(v.conf.FinishTime)
	if err != nil {
		return time.Time{}, err
	}

	// Try Unix timestamp first
	if timestamp, err := strconv.ParseInt(state, 10, 64); err == nil {
		return time.Unix(timestamp, 0), nil
	}

	// Try ISO8601 format
	return time.Parse(time.RFC3339, state)
}

// WakeUp implements api.Resurrector (only if configured)
func (v *HomeAssistant) WakeUp() error {
	if v.wakeupService == "" {
		return api.ErrNotAvailable
	}

	return v.callScript(v.wakeupService)
}

// ChargeEnable implements api.ChargeController (only if configured)
func (v *HomeAssistant) ChargeEnable(enable bool) error {
	service := v.stopService
	if enable {
		service = v.startService
	}

	if service == "" {
		return api.ErrNotAvailable
	}

	return v.callScript(service)
}

// Helper methods
func (v *HomeAssistant) getState(entity string) (string, error) {
	uri := fmt.Sprintf("%s/api/states/%s", v.baseURL, entity)

	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": "Bearer " + v.token,
	})
	if err != nil {
		return "", err
	}

	var res struct {
		State string `json:"state"`
	}

	if err := v.DoJSON(req, &res); err != nil {
		return "", err
	}

	return res.State, nil
}

func (v *HomeAssistant) getFloatSensor(entity string) (float64, error) {
	state, err := v.getState(entity)
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(state, 64)
}

func (v *HomeAssistant) callScript(script string) error {
	uri := fmt.Sprintf("%s/api/services/script/turn_on", v.baseURL)

	data := map[string]interface{}{
		"entity_id": script,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := request.New(http.MethodPost, uri, bytes.NewReader(body), map[string]string{
		"Authorization": "Bearer " + v.token,
		"Content-Type":  "application/json",
	})
	if err != nil {
		return err
	}

	var res interface{}
	return v.DoJSON(req, &res)
}

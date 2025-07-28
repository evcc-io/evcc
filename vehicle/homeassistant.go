// SPDX-License-Identifier: MIT
package vehicle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// ------------------------------------------------------------
// Konfiguration
// ------------------------------------------------------------

type haSensors struct {
	SoC    string `mapstructure:"soc"`
	Range  string `mapstructure:"range"`
	Status string `mapstructure:"status"`
}

type haServices struct {
	Start string `mapstructure:"start_charging"`
	Stop  string `mapstructure:"stop_charging"`
}

type haConfig struct {
	embed     `mapstructure:",squash"`
	Host      string     `mapstructure:"host"`
	Token     string     `mapstructure:"token"`
	Sensors   haSensors  `mapstructure:"sensors"`
	Services  haServices `mapstructure:"services"`
	CacheTime util.Duration `mapstructure:"cache,omitempty"`
}

// ------------------------------------------------------------
// Adapter
// ------------------------------------------------------------

type HomeAssistant struct {
	*embed
	*request.Helper
	conf haConfig
}

func init() { registry.Add("homeassistant", newHAFromConfig) }

func newHAFromConfig(other map[string]any) (api.Vehicle, error) {
	var cc haConfig
	cc.CacheTime = util.Duration{Dur: interval}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.Host == "" || cc.Token == "" || cc.Sensors.SoC == "" {
		return nil, fmt.Errorf("homeassistant: host/token/soc muss gesetzt sein")
	}

	log := util.NewLogger("ha-vehicle").Redact(cc.Token)
	return &HomeAssistant{
		embed:  &cc.embed,
		Helper: request.NewHelper(log),
		conf:   cc,
	}, nil
}

// ------------------------------------------------------------
// Sensor-Helfer
// ------------------------------------------------------------

func (v *HomeAssistant) get(entity string) (string, error) {
	uri := fmt.Sprintf("%s/api/states/%s", strings.TrimSuffix(v.conf.Host, "/"), entity)
	req, err := request.New(http.MethodGet, uri, nil, map[string]string{
		"Authorization": "Bearer " + v.conf.Token,
	})
	if err != nil {
		return "", err
	}
	var resp struct{ State string }
	if err = v.DoJSON(req, &resp); err != nil {
		return "", err
	}
	return resp.State, nil
}

// ------------------------------------------------------------
// Vehicle-Interfaces (lesen)
// ------------------------------------------------------------

func (v *HomeAssistant) SoC() (float64, error) {
	s, err := v.get(v.conf.Sensors.SoC)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}

var _ api.VehicleRange = (*HomeAssistant)(nil)

func (v *HomeAssistant) Range() (int64, error) {
	if v.conf.Sensors.Range == "" {
		return 0, api.ErrNotAvailable
	}
	s, err := v.get(v.conf.Sensors.Range)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}

var _ api.ChargeState = (*HomeAssistant)(nil)

func (v *HomeAssistant) Status() (api.ChargeStatus, error) {
	if v.conf.Sensors.Status == "" {
		return api.StatusNone, api.ErrNotAvailable
	}
	s, err := v.get(v.conf.Sensors.Status)
	if err != nil {
		return api.StatusNone, err
	}
	switch strings.ToLower(s) {
	case "charging", "on", "true":
		return api.StatusC, nil
	case "connected", "ready":
		return api.StatusB, nil
	default:
		return api.StatusA, nil
	}
}

// ------------------------------------------------------------
// Vehicle-Interfaces (schreiben via Script-Service)
// ------------------------------------------------------------

func (v *HomeAssistant) call(script string) error {
	if script == "" {
		return api.ErrNotAvailable
	}
	domain, name, _ := strings.Cut(script, ".")
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

var _ api.VehicleStartCharge = (*HomeAssistant)(nil)
var _ api.VehicleStopCharge  = (*HomeAssistant)(nil)

func (v *HomeAssistant) StartCharge() error { return v.call(v.conf.Services.Start) }
func (v *HomeAssistant) StopCharge()  error { return v.call(v.conf.Services.Stop) }

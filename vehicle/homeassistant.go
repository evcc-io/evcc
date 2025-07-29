// SPDX-License-Identifier: MIT
// homeassistant.go – Vehicle-Adapter für evcc
//
// Einbindung von Fahrzeugdaten aus Home Assistant inklusive
// Ladefreigabe (Start/Stop) via Script-Services.
//
// YAML-Beispiel:
// vehicles:
//   - name: id4
//     type: homeassistant
//     host: http://ha.local:8123
//     token: !secret ha_token
//     sensors:
//       soc: sensor.id4_soc
//       range: sensor.id4_range
//       status: sensor.id4_charging
//     services:
//       start_charging: script.id4_start
//       stop_charging:  script.id4_stop
//     capacity: 77
//
// Änderungen auf Herstellerseite werden ausschließlich
// in den Home-Assistant-Scripts gepflegt.

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
// Konfigurations-Strukturen
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
	embed             `mapstructure:",squash"`
	Host              string     `mapstructure:"host"`  // http://ha:8123
	Token             string     `mapstructure:"token"` // Long-Lived Token
	Sensors           haSensors  `mapstructure:"sensors"`
	Services          haServices `mapstructure:"services"`
	StartChargeScript string     `mapstructure:"start_charging_script"` // optional, analog VW/BMW
	StopChargeScript  string     `mapstructure:"stop_charging_script"`  // optional, analog VW/BMW
}

// -----------------------------
// Adapter-Implementierung
// -----------------------------

type HomeAssistant struct {
	*embed
	*request.Helper
	conf         haConfig
	startScript  string
	stopScript   string
	wakeupScript string
}

// Registrierung beim Start
func init() { registry.Add("homeassistant", newHAFromConfig) }

// Konstruktor ab YAML-Config
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
	// Fallback: falls neue Felder nicht gesetzt, verwende alte Services
	startScript := cc.StartChargeScript
	if startScript == "" {
		startScript = cc.Services.Start
	}
	stopScript := cc.StopChargeScript
	if stopScript == "" {
		stopScript = cc.Services.Stop
	}
	wakeupScript := cc.Services.Wakeup
	return &HomeAssistant{
		embed:        &cc.embed,
		Helper:       request.NewHelper(log),
		conf:         cc,
		startScript:  startScript,
		stopScript:   stopScript,
		wakeupScript: wakeupScript,
	}, nil
}

// -----------------------------
// interne Hilfsfunktionen
// -----------------------------

// ruft /api/states/<entity> ab und liefert .state
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

// führt script.<name> als Service-Call ohne Payload aus
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
// Implementierung API-Interfaces
// -----------------------------

// Soc liefert Ladezustand [%]
func (v *HomeAssistant) Soc() (float64, error) {
	s, err := v.getState(v.conf.Sensors.Soc)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}

// Range liefert Restreichweite [km] (optional)
func (v *HomeAssistant) Range() (int64, error) {
	if v.conf.Sensors.Range == "" {
		return 0, api.ErrNotAvailable
	}
	s, err := v.getState(v.conf.Sensors.Range)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}

// LimitSoc liefert Ziel-Ladezustand [%] (optional)
func (v *HomeAssistant) LimitSoc() (float64, error) {
	if v.conf.Sensors.LimitSoc == "" {
		return 0, api.ErrNotAvailable
	}
	s, err := v.getState(v.conf.Sensors.LimitSoc)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}

// Odometer liefert Kilometerstand [km] (optional)
func (v *HomeAssistant) Odometer() (float64, error) {
	if v.conf.Sensors.Odometer == "" {
		return 0, api.ErrNotAvailable
	}
	s, err := v.getState(v.conf.Sensors.Odometer)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}

// Climater liefert true, wenn Klimatisierung aktiv ist (optional)
func (v *HomeAssistant) Climater() (bool, error) {
	if v.conf.Sensors.Climater == "" {
		return false, api.ErrNotAvailable
	}
	s, err := v.getState(v.conf.Sensors.Climater)
	if err != nil {
		return false, err
	}
	return s == "on" || s == "true" || s == "1" || s == "active", nil
}

// MaxCurrent liefert den maximalen Ladestrom (optional)
func (v *HomeAssistant) MaxCurrent() (int64, error) {
	if v.conf.Sensors.MaxCurrent == "" {
		return 0, api.ErrNotAvailable
	}
	s, err := v.getState(v.conf.Sensors.MaxCurrent)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}

// GetMaxCurrent liefert den aktuell eingestellten maximalen Ladestrom (optional)
func (v *HomeAssistant) GetMaxCurrent() (int64, error) {
	if v.conf.Sensors.GetMaxCurrent == "" {
		return 0, api.ErrNotAvailable
	}
	s, err := v.getState(v.conf.Sensors.GetMaxCurrent)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}

// FinishTime liefert das geplante Ladeende (optional)
func (v *HomeAssistant) FinishTime() (time.Time, error) {
	if v.conf.Sensors.FinishTime == "" {
		return time.Time{}, api.ErrNotAvailable
	}
	s, err := v.getState(v.conf.Sensors.FinishTime)
	if err != nil {
		return time.Time{}, err
	}
	// Erwartet ISO8601-String oder Unix-Timestamp
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(ts, 0), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	return t, err
}

// GetLimitSoc implements the api.SocLimiter interface
func (v *HomeAssistant) GetLimitSoc() (int64, error) {
	if v.conf.Sensors.LimitSoc == "" {
		return 0, api.ErrNotAvailable
	}
	s, err := v.getState(v.conf.Sensors.LimitSoc)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}

// Status liefert evcc-ChargeStatus (optional)
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

// StartCharge löst Script zum Starten aus
func (v *HomeAssistant) StartCharge() error { return v.callScript(v.startScript) }

// StopCharge löst Script zum Stoppen aus
func (v *HomeAssistant) StopCharge() error { return v.callScript(v.stopScript) }

// ChargeEnable schaltet Laden an/aus (UI-Toggle)
func (v *HomeAssistant) ChargeEnable(enable bool) error {
	if enable {
		return v.StartCharge()
	}
	return v.StopCharge()
}

// WakeUp ruft das Wakeup-Script auf (optional)
func (v *HomeAssistant) WakeUp() error {
	if v.wakeupScript == "" {
		return api.ErrNotAvailable
	}
	return v.callScript(v.wakeupScript)
}

// -----------------------------
// Compile-Time-Checks
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

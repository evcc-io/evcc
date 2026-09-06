package charger

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

// ponytail: serialize thermostat commands globally; use keyed locks if concurrent devices need independent I/O.
var thermostatMu sync.Mutex

type thermostatBoost struct {
	Baseline  float64 `json:"baseline"`
	Target    float64 `json:"target"`
	Confirmed bool    `json:"confirmed"`
	Hold      bool    `json:"hold"`
	Restoring bool    `json:"restoring"`
}

// HomeAssistantThermostat temporarily raises a single heating target.
type HomeAssistantThermostat struct {
	*embed
	implement.Caps
	conn   *homeassistant.Connection
	entity string
	boost  float64
}

func init() {
	registry.Add("homeassistant-thermostat", NewHomeAssistantThermostatFromConfig)
}

// NewHomeAssistantThermostatFromConfig creates a thermostat boost controller.
func NewHomeAssistantThermostatFromConfig(other map[string]any) (api.Charger, error) {
	var cc struct {
		embed                `mapstructure:",squash"`
		homeassistant.Config `mapstructure:",squash"`
		Entity               string
		Boost                float64
		Power                string
	}
	cc.Icon_ = "heatpump"
	cc.Features_ = []api.Feature{api.Continuous, api.Heating, api.IntegratedDevice, api.SwitchDevice}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if domain, name, ok := strings.Cut(cc.Entity, "."); !ok || domain != "climate" || name == "" || strings.ContainsAny(name, "/?# ") {
		return nil, errors.New("entity must be a climate entity")
	}
	if !thermostatFinite(cc.Boost) || cc.Boost <= 0 {
		return nil, errors.New("boost must be a positive temperature increase")
	}
	conn, err := cc.Config.NewConnection(util.NewLogger("ha-thermostat"))
	if err != nil {
		return nil, err
	}
	c := &HomeAssistantThermostat{
		embed: &cc.embed, Caps: implement.New(), conn: conn, entity: cc.Entity, boost: cc.Boost,
	}
	if cc.Power != "" {
		implement.Has(c, implement.Meter(func() (float64, error) {
			state, err := conn.GetState(cc.Power)
			if err != nil {
				return 0, err
			}
			unit := state.Attributes.UnitOfMeasurement
			if unit != "W" && unit != "kW" {
				return 0, errors.New("thermostat power sensor must use W or kW")
			}
			power, err := strconv.ParseFloat(state.State, 64)
			if unit == "kW" {
				power *= 1e3
			}
			if err == nil && (!thermostatFinite(power) || power < 0) {
				err = errors.New("invalid thermostat power")
			}
			return power, err
		}))
	}
	return c, nil
}

func thermostatFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func thermostatEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

func (c *HomeAssistantThermostat) key() string {
	return fmt.Sprintf("thermostat:%x", sha256.Sum256([]byte(c.conn.URI()+"\x00"+c.entity)))
}

func (c *HomeAssistantThermostat) read() (homeassistant.ClimateStateResponse, error) {
	var config struct {
		UnitSystem struct {
			Temperature string `json:"temperature"`
		} `json:"unit_system"`
	}
	if err := c.conn.GetJSON(c.conn.URI()+"/api/config", &config); err != nil {
		return homeassistant.ClimateStateResponse{}, err
	}
	if config.UnitSystem.Temperature != "°C" {
		return homeassistant.ClimateStateResponse{}, errors.New("thermostat requires Home Assistant Celsius units")
	}
	s, err := c.conn.GetClimateState(c.entity)
	if err != nil {
		return s, err
	}
	a := s.Attributes
	if a.UnitOfMeasurement != "" && a.UnitOfMeasurement != "°C" {
		return s, errors.New("thermostat requires Celsius temperature attributes")
	}
	if s.State != "heat" && s.State != "off" {
		return s, fmt.Errorf("unsupported thermostat mode: %s", s.State)
	}
	if a.SupportedFeatures&1 == 0 || a.Temperature == nil || a.MinTemp == nil || a.MaxTemp == nil {
		return s, errors.New("thermostat requires a single writable temperature and limits")
	}
	if !thermostatFinite(*a.MinTemp) || !thermostatFinite(*a.MaxTemp) || *a.MinTemp >= *a.MaxTemp {
		return s, errors.New("invalid thermostat temperature limits")
	}
	return s, thermostatTarget(s, *a.Temperature)
}

func thermostatTarget(s homeassistant.ClimateStateResponse, target float64) error {
	a := s.Attributes
	if !thermostatFinite(target) || target < *a.MinTemp || target > *a.MaxTemp {
		return errors.New("thermostat target outside temperature limits")
	}
	if step := a.TargetTempStep; step != nil {
		if !thermostatFinite(*step) || *step <= 0 || !thermostatEqual((target-*a.MinTemp) / *step, math.Round((target-*a.MinTemp) / *step)) {
			return errors.New("thermostat target does not match temperature step")
		}
	}
	return nil
}

func (c *HomeAssistantThermostat) saved() (*thermostatBoost, error) {
	var saved thermostatBoost
	if err := settings.Json(c.key(), &saved); errors.Is(err, settings.ErrNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if !thermostatFinite(saved.Baseline) || !thermostatFinite(saved.Target) || saved.Target <= saved.Baseline {
		return nil, errors.New("invalid saved thermostat boost")
	}
	return &saved, nil
}

func (c *HomeAssistantThermostat) save(saved *thermostatBoost) error {
	if db.Instance == nil {
		return errors.New("thermostat boost requires a persistent database")
	}
	var databases []struct{ Name, File string }
	if err := db.Instance.Raw("PRAGMA database_list").Scan(&databases).Error; err != nil {
		return err
	}
	if !slices.ContainsFunc(databases, func(d struct{ Name, File string }) bool { return d.Name == "main" && d.File != "" }) {
		return errors.New("thermostat boost requires a persistent database")
	}
	key := c.key()
	previous, err := settings.String(key)
	if err != nil && !errors.Is(err, settings.ErrNotFound) {
		return err
	}
	if err := settings.SetJson(key, saved); err != nil {
		return err
	}
	if err := settings.Persist(); err != nil {
		settings.SetString(key, previous)
		return fmt.Errorf("persist thermostat boost: %w", err)
	}
	return nil
}

// observe records acknowledgements without sending thermostat commands.
func (c *HomeAssistantThermostat) observe(s homeassistant.ClimateStateResponse) (*thermostatBoost, error) {
	saved, err := c.saved()
	if err != nil || saved == nil || saved.Restoring {
		return saved, err
	}
	before := *saved
	target := *s.Attributes.Temperature
	if thermostatEqual(target, saved.Target) {
		saved.Confirmed = true
	}
	switch {
	case saved.Hold:
	case s.State != "heat":
		saved.Hold = true
	case thermostatEqual(target, saved.Target):
	case saved.Confirmed || !thermostatEqual(target, saved.Baseline):
		saved.Hold = true
	}
	if *saved != before {
		err = c.save(saved)
	}
	return saved, err
}

var _ api.Charger = (*HomeAssistantThermostat)(nil)

// Status implements api.ChargeState; compressor activity is measured separately.
func (c *HomeAssistantThermostat) Status() (api.ChargeStatus, error) {
	thermostatMu.Lock()
	defer thermostatMu.Unlock()
	s, err := c.read()
	if err != nil {
		return api.StatusNone, err
	}
	saved, err := c.observe(s)
	if err != nil {
		return api.StatusNone, err
	}
	if saved != nil && saved.Confirmed && !saved.Hold && !saved.Restoring {
		return api.StatusC, nil
	}
	return api.StatusB, nil
}

// Enabled includes pending restoration and manual hold until the next Normal request.
func (c *HomeAssistantThermostat) Enabled() (bool, error) {
	thermostatMu.Lock()
	defer thermostatMu.Unlock()
	s, err := c.read()
	if err != nil {
		return false, err
	}
	saved, err := c.observe(s)
	return saved != nil, err
}

// Enable applies one bounded increase or restores the captured target.
func (c *HomeAssistantThermostat) Enable(enable bool) error {
	thermostatMu.Lock()
	defer thermostatMu.Unlock()
	s, err := c.read()
	if err != nil {
		return err
	}
	saved, err := c.observe(s)
	if err != nil {
		return err
	}
	if enable {
		if saved != nil {
			if saved.Restoring {
				return errors.New("thermostat restoration pending; select Off to finish")
			}
			return nil
		}
		if s.State != "heat" {
			return errors.New("thermostat must already be in heat mode")
		}
		baseline := *s.Attributes.Temperature
		target := baseline + c.boost
		if target <= baseline || thermostatEqual(target, baseline) {
			return errors.New("boost is below thermostat comparison precision")
		}
		if err := thermostatTarget(s, target); err != nil {
			return err
		}
		saved = &thermostatBoost{Baseline: baseline, Target: target}
		if err := c.save(saved); err != nil {
			return err
		}
		return c.setTemperature(target)
	}
	if saved == nil {
		return nil
	}
	target := *s.Attributes.Temperature
	if !saved.Confirmed {
		return errors.New("thermostat boost not yet observed; restoration remains pending")
	}
	if saved.Restoring {
		if thermostatEqual(target, saved.Baseline) {
			return settings.Delete(c.key())
		}
		if !thermostatEqual(target, saved.Target) || s.State != "heat" {
			return errors.New("thermostat restoration unconfirmed; preserving the observed target")
		}
	} else if saved.Hold {
		if thermostatEqual(target, saved.Target) {
			return errors.New("boost observed after manual change; restore the thermostat manually")
		}
		return settings.Delete(c.key())
	}
	if err := thermostatTarget(s, saved.Baseline); err != nil {
		return err
	}
	saved.Restoring = true
	if err := c.save(saved); err != nil {
		return err
	}
	if err := c.setTemperature(saved.Baseline); err != nil {
		return err
	}
	s, err = c.read()
	if err != nil {
		return err
	}
	if !thermostatEqual(*s.Attributes.Temperature, saved.Baseline) {
		return errors.New("thermostat restoration not yet observed")
	}
	return settings.Delete(c.key())
}

func (c *HomeAssistantThermostat) setTemperature(target float64) error {
	return c.conn.CallService("climate", "set_temperature", map[string]any{
		"entity_id": c.entity, "temperature": target,
	})
}

// MaxCurrent implements api.Charger; this device cannot control electrical current.
func (c *HomeAssistantThermostat) MaxCurrent(int64) error { return nil }

var _ api.Battery = (*HomeAssistantThermostat)(nil)

// Soc implements api.Battery as the measured room temperature in Celsius.
func (c *HomeAssistantThermostat) Soc() (float64, error) {
	s, err := c.read()
	if err != nil {
		return 0, err
	}
	if s.Attributes.CurrentTemperature == nil || !thermostatFinite(*s.Attributes.CurrentTemperature) {
		return 0, api.ErrNotAvailable
	}
	return *s.Attributes.CurrentTemperature, nil
}

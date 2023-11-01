package server

import (
	"errors"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
)

const (
	// typeTemplate is the updatable configuration type
	typeTemplate = "template"

	// masked indicates a masked config parameter value
	masked = "***"
)

var (
	dirty bool
	mu    sync.Mutex
)

// ConfigDirty returns the dirty flag
func ConfigDirty() bool {
	mu.Lock()
	defer mu.Unlock()

	return dirty
}

// setConfigDirty sets the dirty flag indicating that a restart is required
func setConfigDirty() {
	mu.Lock()
	defer mu.Unlock()

	dirty = true
}

func templateForConfig(class templates.Class, conf map[string]any) (templates.Template, error) {
	typ, ok := conf[typeTemplate].(string)
	if !ok {
		return templates.Template{}, errors.New("config template not found")
	}

	return templates.ByName(class, typ)
}

func sanitizeMasked(class templates.Class, conf map[string]any) (map[string]any, error) {
	tmpl, err := templateForConfig(class, conf)
	if err != nil {
		return nil, err
	}

	res := make(map[string]any, len(conf))

	for k, v := range conf {
		if i, p := tmpl.ParamByName(k); i >= 0 && p.IsMasked() {
			v = masked
		}

		res[k] = v
	}

	return res, nil
}

func mergeMasked(class templates.Class, conf, old map[string]any) (map[string]any, error) {
	tmpl, err := templateForConfig(class, conf)
	if err != nil {
		return nil, err
	}

	res := make(map[string]any, len(conf))

	for k, v := range conf {
		if i, p := tmpl.ParamByName(k); i >= 0 && p.IsMasked() && v == masked {
			v = old[k]
		}

		res[k] = v
	}

	return res, nil
}

func deviceInstanceFromMergedConfig[T any](id int, class templates.Class, conf map[string]any, newFromConf func(string, map[string]any) (T, error), h config.Handler[T]) (config.Device[T], T, map[string]any, error) {
	var zero T

	dev, err := h.ByName(config.NameForID(id))
	if err != nil {
		return nil, zero, nil, err
	}

	merged, err := mergeMasked(class, conf, dev.Config().Other)
	if err != nil {
		return nil, zero, nil, err
	}

	instance, err := newFromConf(typeTemplate, merged)

	return dev, instance, merged, err
}

type testResult = struct {
	Value any    `json:"value"`
	Error string `json:"error"`
}

// testInstance tests the given instance similar to dump
// TODO refactor together with dump
func testInstance(instance any) map[string]testResult {
	res := make(map[string]testResult)

	makeResult := func(val any, err error) testResult {
		res := testResult{Value: val}
		if err != nil {
			res.Error = err.Error()
		}
		return res
	}

	if dev, ok := instance.(api.Meter); ok {
		val, err := dev.CurrentPower()
		res["power"] = makeResult(val, err)
	}

	if dev, ok := instance.(api.MeterEnergy); ok {
		val, err := dev.TotalEnergy()
		res["energy"] = makeResult(val, err)
	}

	if dev, ok := instance.(api.Battery); ok {
		val, err := dev.Soc()
		res["soc"] = makeResult(val, err)
	}

	if dev, ok := instance.(api.VehicleOdometer); ok {
		val, err := dev.Odometer()
		res["odometer"] = makeResult(val, err)
	}

	if dev, ok := instance.(api.BatteryCapacity); ok {
		val := dev.Capacity()
		res["capacity"] = makeResult(val, nil)
	}

	if dev, ok := instance.(api.PhaseCurrents); ok {
		i1, i2, i3, err := dev.Currents()
		res["phaseCurrents"] = makeResult([]float64{i1, i2, i3}, err)
	}

	if dev, ok := instance.(api.PhaseVoltages); ok {
		u1, u2, u3, err := dev.Voltages()
		res["phaseVoltages"] = makeResult([]float64{u1, u2, u3}, err)
	}

	if dev, ok := instance.(api.PhasePowers); ok {
		p1, p2, p3, err := dev.Powers()
		res["phasePowers"] = makeResult([]float64{p1, p2, p3}, err)
	}

	if dev, ok := instance.(api.ChargeState); ok {
		val, err := dev.Status()
		res["chargeStatus"] = makeResult(val, err)
	}

	if dev, ok := instance.(api.Charger); ok {
		val, err := dev.Enabled()
		res["enabled"] = makeResult(val, err)
	}

	if dev, ok := instance.(api.ChargeRater); ok {
		val, err := dev.ChargedEnergy()
		res["chargedEnergy"] = makeResult(val, err)
	}

	if dev, ok := instance.(api.VehicleRange); ok {
		val, err := dev.Range()
		res["range"] = makeResult(val, err)
	}

	if dev, ok := instance.(api.SocLimiter); ok {
		val, err := dev.TargetSoc()
		res["socLimit"] = makeResult(val, err)
	}

	return res
}

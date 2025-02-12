package server

import (
	"context"
	"errors"
	"slices"
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

type newFromConfFunc[T any] func(context.Context, string, map[string]any) (T, error)

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

func deviceInstanceFromMergedConfig[T any](id int, class templates.Class, conf map[string]any, newFromConf newFromConfFunc[T], h config.Handler[T]) (config.Device[T], T, map[string]any, error) {
	var zero T

	dev, err := h.ByName(config.NameForID(id))
	if err != nil {
		return nil, zero, nil, err
	}

	merged, err := mergeMasked(class, conf, dev.Config().Other)
	if err != nil {
		return nil, zero, nil, err
	}

	instance, err := newFromConf(context.TODO(), typeTemplate, merged)

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

	makeResult := func(key string, val any, err error) {
		tr := testResult{Value: val}
		if err != nil {
			if errors.Is(err, api.ErrNotAvailable) {
				return
			}
			tr.Error = err.Error()
		}
		res[key] = tr
	}

	if dev, ok := instance.(api.Meter); ok {
		val, err := dev.CurrentPower()
		makeResult("power", val, err)
	}

	if dev, ok := instance.(api.MeterEnergy); ok {
		val, err := dev.TotalEnergy()
		makeResult("energy", val, err)
	}

	if dev, ok := instance.(api.Battery); ok {
		val, err := dev.Soc()
		key := "soc"
		if fd, ok := instance.(api.FeatureDescriber); ok && slices.Contains(fd.Features(), api.Heating) {
			key = "temp"
		}
		makeResult(key, val, err)
	}

	if _, ok := instance.(api.BatteryController); ok {
		makeResult("controllable", true, nil)
	}

	if dev, ok := instance.(api.VehicleOdometer); ok {
		val, err := dev.Odometer()
		makeResult("odometer", val, err)
	}

	if dev, ok := instance.(api.BatteryCapacity); ok {
		val := dev.Capacity()
		makeResult("capacity", val, nil)
	}

	if dev, ok := instance.(api.PhaseCurrents); ok {
		i1, i2, i3, err := dev.Currents()
		makeResult("phaseCurrents", []float64{i1, i2, i3}, err)
	}

	if dev, ok := instance.(api.PhaseVoltages); ok {
		u1, u2, u3, err := dev.Voltages()
		makeResult("phaseVoltages", []float64{u1, u2, u3}, err)
	}

	if dev, ok := instance.(api.PhasePowers); ok {
		p1, p2, p3, err := dev.Powers()
		makeResult("phasePowers", []float64{p1, p2, p3}, err)
	}

	if dev, ok := instance.(api.ChargeState); ok {
		val, err := dev.Status()
		makeResult("chargeStatus", val, err)
	}

	if dev, ok := instance.(api.Charger); ok {
		val, err := dev.Enabled()
		makeResult("enabled", val, err)
	}

	if dev, ok := instance.(api.ChargeRater); ok {
		val, err := dev.ChargedEnergy()
		makeResult("chargedEnergy", val, err)
	}

	if _, ok := instance.(api.PhaseSwitcher); ok {
		makeResult("phases1p3p", true, nil)
	}

	if cc, ok := instance.(api.PhaseDescriber); ok && cc.Phases() == 1 {
		makeResult("singlePhase", true, nil)
	}

	if dev, ok := instance.(api.VehicleRange); ok {
		val, err := dev.Range()
		makeResult("range", val, err)
	}

	if dev, ok := instance.(api.SocLimiter); ok {
		val, err := dev.GetLimitSoc()
		makeResult("vehicleLimitSoc", val, err)
	}

	if dev, ok := instance.(api.Identifier); ok {
		val, err := dev.Identify()
		makeResult("identifier", val, err)
	}

	return res
}

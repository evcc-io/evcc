package server

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/go-viper/mapstructure/v2"
	"github.com/samber/lo"
)

const (
	// typeTemplate is the updatable configuration type
	typeTemplate = "template"

	// masked indicates a masked config parameter value
	masked = "***"
)

type configReq struct {
	config.Properties `json:",inline" mapstructure:",squash"`
	Other             map[string]any `json:",inline" mapstructure:",remain"`
}

// TODO get rid of this 2-pass unmarshal once https://github.com/golang/go/issues/71497 is implemented
func (c *configReq) UnmarshalJSON(data []byte) error {
	var res map[string]any
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}

	var cr configReq
	if err := util.DecodeOther(res, &cr); err != nil {
		return err
	}

	*c = cr
	return nil
}

func propsToMap(props config.Properties) (map[string]any, error) {
	res := make(map[string]any)
	if err := mapstructure.Decode(props, &res); err != nil {
		return nil, err
	}

	return lo.PickBy(res, func(k string, v any) bool {
		if k == "Type" || v.(string) == "" {
			return false
		}
		return true
	}), nil
}

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

func startDeviceTimeout() (context.Context, context.CancelFunc, chan struct{}) {
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		select {
		case <-time.After(10 * time.Second):
			// timeout - cancel context
			cancel()
		case <-done:
			// success
		}
	}()

	return ctx, cancel, done
}

func deviceInstanceFromMergedConfig[T any](ctx context.Context, id int, class templates.Class, req configReq, newFromConf newFromConfFunc[T], h config.Handler[T]) (config.Device[T], T, map[string]any, error) {
	var zero T

	dev, err := h.ByName(config.NameForID(id))
	if err != nil {
		return nil, zero, nil, err
	}

	conf := dev.Config()

	merged, err := mergeMasked(class, req.Other, conf.Other)
	if err != nil {
		return nil, zero, nil, err
	}

	instance, err := newFromConf(ctx, conf.Type, merged)

	return dev, instance, merged, err
}

type testResult = struct {
	Value any    `json:"value"`
	Error string `json:"error"`
}

func hasFeature(instance any, f api.Feature) bool {
	fd, ok := instance.(api.FeatureDescriber)
	return ok && slices.Contains(fd.Features(), f)
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
		if hasFeature(instance, api.Heating) {
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
		key := "vehicleLimitSoc"
		if hasFeature(instance, api.Heating) {
			key = "heaterTempLimit"
		}
		makeResult(key, val, err)
	}

	if dev, ok := instance.(api.Identifier); ok {
		val, err := dev.Identify()
		makeResult("identifier", val, err)
	}

	return res
}

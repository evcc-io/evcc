package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"dario.cat/mergo"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/go-viper/mapstructure/v2"
	"github.com/samber/lo"
	"go.yaml.in/yaml/v4"
)

const (
	typeTemplate = "template" // typeTemplate is the updatable configuration type
	masked       = "***"      // masked indicates a masked config parameter value
)

var (
	customTypes = []string{"custom", "template", "heatpump", "switchsocket", "sgready", "sgready-relay"}
)

type configReq struct {
	config.Properties `json:",inline" mapstructure:",squash"`
	Yaml              string
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

func (c *configReq) Serialise() map[string]any {
	if c.Yaml != "" {
		return map[string]any{
			"yaml": c.Yaml,
		}
	}
	return c.Other
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

func filterValidTemplateParams(tmpl *templates.Template, conf map[string]any) map[string]any {
	res := make(map[string]any)

	// check if template has modbus capability
	hasModbus := len(tmpl.ModbusChoices()) > 0

	for k, v := range conf {
		if k == "template" {
			res[k] = v
			continue
		}

		// preserve modbus fields if template supports modbus
		if hasModbus && slices.Contains(templates.ModbusParams, k) {
			res[k] = v
			continue
		}

		if i, _ := tmpl.ParamByName(k); i >= 0 {
			res[k] = v
		}
	}

	return res
}

func sanitizeMasked(class templates.Class, conf map[string]any, hidePrivate bool) (map[string]any, error) {
	tmpl, err := templateForConfig(class, conf)
	if err != nil {
		return nil, err
	}

	res := make(map[string]any, len(conf))

	for k, v := range conf {
		if i, p := tmpl.ParamByName(k); i >= 0 {
			if p.IsMasked() {
				v = masked
			} else if hidePrivate && p.IsPrivate() {
				v = masked
			}
		}

		res[k] = v
	}

	return filterValidTemplateParams(&tmpl, res), nil
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

	return filterValidTemplateParams(&tmpl, res), nil
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

	// TODO merge custom config
	if req.Yaml != "" {
		instance, err := newFromConf(ctx, conf.Type, req.Other)
		return dev, instance, req.Serialise(), err
	}

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

	if hasFeature(instance, api.Heating) {
		makeResult("heating", true, nil)
	}

	if hasFeature(instance, api.IntegratedDevice) {
		makeResult("integratedDevice", true, nil)
	}

	if dev, ok := instance.(api.IconDescriber); ok && dev.Icon() != "" {
		makeResult("icon", dev.Icon(), nil)
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

	if dev, ok := instance.(api.Dimmer); ok {
		val, err := dev.Dimmed()
		makeResult("dimmed", val, err)
	}

	if dev, ok := instance.(api.Identifier); ok {
		val, err := dev.Identify()
		makeResult("identifier", val, err)
	}

	return res
}

// mergeMaskedAny similar to mergeMasked but for interfaces
func mergeMaskedAny(old, new any) error {
	return mergo.Merge(new, old, mergo.WithTransformers(&maskedTransformer{}))
}

type maskedTransformer struct{}

func (maskedTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	// Only provide transformer for booleans to prevent them from being merged
	if typ.Kind() == reflect.Bool {
		return func(dst, src reflect.Value) error {
			// Keep dst value, don't merge
			return nil
		}
	}

	// Preserve slices/arrays: in general don't merge them into dst, but try to merge elements when both slices are non-empty
	if typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
		// If the slice elements are structs, attempt element-wise merge for non-empty slices
		if typ.Elem().Kind() == reflect.Struct {
			return func(dst, src reflect.Value) error {
				if !dst.IsValid() || dst.IsNil() || !src.IsValid() || src.IsNil() {
					return nil
				}

				if dst.Len() == 0 || src.Len() == 0 {
					// Keep dst value, don't merge
					return nil
				}

				// Merge element-wise up to the shorter length
				n := min(src.Len(), dst.Len())

				for i := range n {
					de := dst.Index(i)
					se := src.Index(i)
					// If addressable, merge using mergo to apply transformers
					if de.CanAddr() {
						if err := mergo.Merge(de.Addr().Interface(), se.Interface(), mergo.WithTransformers(&maskedTransformer{})); err != nil {
							return err
						}
					}
				}

				return nil
			}
		}

		return func(dst, src reflect.Value) error {
			// Keep dst value, don't merge
			return nil
		}
	}

	// Provide transformer for maps[string]any to detect masked values and restore them
	if typ.Kind() == reflect.Map && typ.Key().Kind() == reflect.String {
		return func(dst, src reflect.Value) error {
			if !dst.IsValid() || dst.IsNil() || !src.IsValid() || src.IsNil() {
				return nil
			}

			// Build lowercase lookup for src keys
			srcKeys := make(map[string]reflect.Value)
			for _, k := range src.MapKeys() {
				srcKeys[strings.ToLower(k.String())] = k
			}

			// Iterate dst keys and replace masked entries with src values using case-insensitive match
			for _, k := range dst.MapKeys() {
				dk := k.String()
				dv := dst.MapIndex(k)

				// Unwrap interface values
				if dv.Kind() == reflect.Interface && dv.IsValid() && !dv.IsNil() {
					dv = dv.Elem()
				}

				if dv.IsValid() && dv.Kind() == reflect.String && dv.String() == masked {
					if sk, ok := srcKeys[strings.ToLower(dk)]; ok {
						// Remove the masked dst key
						dst.SetMapIndex(k, reflect.Value{})
						// Set dst[srcKey] = src[srcKey]
						sv := src.MapIndex(sk)
						if sv.IsValid() {
							dst.SetMapIndex(sk, sv)
						}
					}
				}
			}

			return nil
		}
	}

	// Only provide transformer for strings
	if typ.Kind() != reflect.String {
		return nil
	}

	return func(dst, src reflect.Value) error {
		if dst.String() == masked {
			dst.Set(src)
		}

		return nil
	}
}

func decodeDeviceConfig(r io.Reader) (configReq, error) {
	var res configReq

	if err := json.NewDecoder(r).Decode(&res); err != nil {
		return configReq{}, err
	}

	if res.Yaml == "" {
		return res, nil
	}

	if !slices.ContainsFunc(customTypes, func(s string) bool {
		return strings.EqualFold(res.Type, s)
	}) {
		return configReq{}, errors.New("invalid config: yaml only allowed for types " + strings.Join(customTypes, ", "))
	}

	if len(res.Other) != 0 {
		return configReq{}, errors.New("invalid config: cannot mix yaml and other")
	}

	if err := yaml.Unmarshal([]byte(res.Yaml), &res.Other); err != nil && err != io.EOF {
		return configReq{}, err
	}

	return res, nil
}

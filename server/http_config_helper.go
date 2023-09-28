package server

import (
	"errors"
	"sync"

	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
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

func deviceInstanceFromMergedConfig[T any](id int, class templates.Class, conf map[string]any, newFromConf func(string, map[string]any) (T, error), h config.Handler[T]) (config.Device[T], T, error) {
	var zero T

	dev, err := h.ByName(config.NameForID(id))
	if err != nil {
		return nil, zero, err
	}

	merged, err := mergeMasked(class, conf, dev.Config().Other)
	if err != nil {
		return nil, zero, err
	}

	instance, err := newFromConf(typeTemplate, merged)

	return dev, instance, err
}

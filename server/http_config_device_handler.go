package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"dario.cat/mergo"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/gorilla/mux"
)

func devicesConfig[T any](class templates.Class, h config.Handler[T]) ([]map[string]any, error) {
	var res []map[string]any

	for _, dev := range h.Devices() {
		dc, err := deviceConfigMap(class, dev)
		if err != nil {
			return nil, err
		}

		res = append(res, dc)
	}

	return res, nil
}

// devicesConfigHandler returns a device configurations by class
func devicesConfigHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := templates.ClassString(vars["class"])

	// TODO exclude loadpoints here
	if err == nil && class == templates.Loadpoint {
		err = fmt.Errorf("invalid class %s", class)
	}

	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	var res []map[string]any

	switch class {
	case templates.Meter:
		res, err = devicesConfig(class, config.Meters())

	case templates.Charger:
		res, err = devicesConfig(class, config.Chargers())

	case templates.Vehicle:
		res, err = devicesConfig(class, config.Vehicles())

	case templates.Circuit:
		res, err = devicesConfig(class, config.Circuits())
	}

	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	jsonResult(w, res)
}

func deviceConfigMap[T any](class templates.Class, dev config.Device[T]) (map[string]any, error) {
	conf := dev.Config()

	dc := map[string]any{
		"name": conf.Name,
	}
	if conf.Type != "" {
		dc["type"] = conf.Type
	}

	if configurable, ok := dev.(config.ConfigurableDevice[T]); ok {
		// from database
		dc["id"] = configurable.ID()

		props, err := propsToMap(configurable.Properties())
		if err != nil {
			return nil, err
		}

		if err := mergo.Merge(&dc, props); err != nil {
			return nil, err
		}

		if conf.Type == typeTemplate {
			// template device, mask config
			params, err := sanitizeMasked(class, conf.Other)
			if err != nil {
				return nil, err
			}
			dc["config"] = params
		} else {
			// custom device, no masking
			dc["config"] = conf.Other
		}
	}

	if dc["config"] == nil {
		// add title if available
		config := make(map[string]any)
		if title, ok := conf.Other["title"].(string); ok {
			config["title"] = title
		}
		// add icon if available
		if icon, ok := conf.Other["icon"].(string); ok {
			config["icon"] = icon
		}
		if len(config) > 0 {
			dc["config"] = config
		}
	}

	return dc, nil
}

func deviceConfig[T any](class templates.Class, id int, h config.Handler[T]) (map[string]any, error) {
	dev, err := h.ByName(config.NameForID(id))
	if err != nil {
		return nil, err
	}

	return deviceConfigMap(class, dev)
}

// deviceConfigHandler returns a device configuration by class
func deviceConfigHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := templates.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	var res map[string]any

	switch class {
	case templates.Meter:
		res, err = deviceConfig(class, id, config.Meters())

	case templates.Charger:
		res, err = deviceConfig(class, id, config.Chargers())

	case templates.Vehicle:
		res, err = deviceConfig(class, id, config.Vehicles())

	case templates.Circuit:
		res, err = deviceConfig(class, id, config.Circuits())
	}

	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// TODO return application/yaml content type if type != template

	jsonResult(w, res)
}

func deviceStatus[T any](name string, h config.Handler[T]) (T, error) {
	dev, err := h.ByName(name)
	if err != nil {
		var zero T
		return zero, err
	}

	return dev.Instance(), nil
}

// deviceStatusHandler returns the device test status by class
func deviceStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := templates.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	name := vars["name"]

	var instance any

	switch class {
	case templates.Meter:
		instance, err = deviceStatus(name, config.Meters())

	case templates.Charger:
		instance, err = deviceStatus(name, config.Chargers())

	case templates.Vehicle:
		instance, err = deviceStatus(name, config.Vehicles())

	case templates.Circuit:
		instance, err = deviceStatus(name, config.Circuits())
	}

	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	jsonResult(w, testInstance(instance))
}

func newDevice[T any](ctx context.Context, class templates.Class, req configReq, newFromConf newFromConfFunc[T], h config.Handler[T]) (*config.Config, error) {
	instance, err := newFromConf(ctx, req.Type, req.Other)
	if err != nil {
		return nil, err
	}

	conf, err := config.AddConfig(class, req.Serialise(), config.WithProperties(req.Properties))
	if err != nil {
		return nil, err
	}

	return &conf, h.Add(config.NewConfigurableDevice(&conf, instance))
}

// newDeviceHandler creates a new device by class
func newDeviceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := templates.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	req, err := decodeDeviceConfig(r.Body)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	var conf *config.Config
	ctx, cancel, done := startDeviceTimeout()

	switch class {
	case templates.Charger:
		conf, err = newDevice(ctx, class, req, charger.NewFromConfig, config.Chargers())

	case templates.Meter:
		conf, err = newDevice(ctx, class, req, meter.NewFromConfig, config.Meters())

	case templates.Vehicle:
		conf, err = newDevice(ctx, class, req, vehicle.NewFromConfig, config.Vehicles())

	case templates.Circuit:
		conf, err = newDevice(ctx, class, req, func(ctx context.Context, _ string, other map[string]interface{}) (api.Circuit, error) {
			return circuit.NewFromConfig(ctx, util.NewLogger("circuit"), other)
		}, config.Circuits())
	}

	if err != nil {
		cancel()
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// prevent context from being cancelled
	close(done)

	setConfigDirty()

	res := struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{
		ID:   conf.ID,
		Name: config.NameForID(conf.ID),
	}

	jsonResult(w, res)
}

func updateDevice[T any](ctx context.Context, id int, class templates.Class, req configReq, newFromConf newFromConfFunc[T], h config.Handler[T]) error {
	dev, instance, merged, err := deviceInstanceFromMergedConfig(ctx, id, class, req, newFromConf, h)
	if err != nil {
		return err
	}

	configurable, ok := dev.(config.ConfigurableDevice[T])
	if !ok {
		return errors.New("not configurable")
	}

	return configurable.Update(merged, instance, config.WithProperties(req.Properties))
}

// updateDeviceHandler updates database device's configuration by class
func updateDeviceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := templates.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	req, err := decodeDeviceConfig(r.Body)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	ctx, cancel, done := startDeviceTimeout()

	switch class {
	case templates.Charger:
		err = updateDevice(ctx, id, class, req, charger.NewFromConfig, config.Chargers())

	case templates.Meter:
		err = updateDevice(ctx, id, class, req, meter.NewFromConfig, config.Meters())

	case templates.Vehicle:
		err = updateDevice(ctx, id, class, req, vehicle.NewFromConfig, config.Vehicles())

	case templates.Circuit:
		err = updateDevice(ctx, id, class, req, func(ctx context.Context, _ string, other map[string]interface{}) (api.Circuit, error) {
			return circuit.NewFromConfig(ctx, util.NewLogger("circuit"), other)
		}, config.Circuits())
	}

	setConfigDirty()

	if err != nil {
		cancel()
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// prevent context from being cancelled
	close(done)

	res := struct {
		ID int `json:"id"`
	}{
		ID: id,
	}

	jsonResult(w, res)
}

func configurableDevice[T any](name string, h config.Handler[T]) (config.ConfigurableDevice[T], error) {
	dev, err := h.ByName(name)
	if err != nil {
		return nil, err
	}

	configurable, ok := dev.(config.ConfigurableDevice[T])
	if !ok {
		return nil, errors.New("not configurable")
	}

	return configurable, nil
}

func deleteDevice[T any](id int, h config.Handler[T]) error {
	name := config.NameForID(id)

	configurable, err := configurableDevice(name, h)
	if err != nil {
		return err
	}

	if err := configurable.Delete(); err != nil {
		return err
	}

	return h.Delete(name)
}

// cleanupSiteMeterRef removes a meter reference from site configuration
func cleanupSiteMeterRef(name string, get func() []string, set func([]string)) {
	var res []string
	refs := get()
	for _, ref := range refs {
		if ref != name {
			res = append(res, ref)
		}
	}
	if len(refs) != len(res) {
		set(res)
	}
}

// deleteDeviceHandler deletes a device from database by class
func deleteDeviceHandler(site site.API) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		h := config.Loadpoints()

		vars := mux.Vars(r)

		class, err := templates.ClassString(vars["class"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		switch class {
		case templates.Charger:
			err = deleteDevice(id, config.Chargers())

			// cleanup references
			for _, dev := range h.Devices() {
				lp := dev.Instance()
				if lp.GetChargerRef() == config.NameForID(id) {
					lp.SetChargerRef("")
				}
			}

		case templates.Meter:
			err = deleteDevice(id, config.Meters())

			// cleanup references
			name := config.NameForID(id)

			if site.GetGridMeterRef() == name {
				site.SetGridMeterRef("")
			}

			for _, fun := range []struct {
				get func() []string
				set func([]string)
			}{
				{site.GetPVMeterRefs, site.SetPVMeterRefs},
				{site.GetBatteryMeterRefs, site.SetBatteryMeterRefs},
				{site.GetAuxMeterRefs, site.SetAuxMeterRefs},
				{site.GetExtMeterRefs, site.SetExtMeterRefs},
			} {
				cleanupSiteMeterRef(name, fun.get, fun.set)
			}

			for _, dev := range h.Devices() {
				lp := dev.Instance()
				if lp.GetMeterRef() == name {
					lp.SetMeterRef("")
				}
			}

		case templates.Vehicle:
			err = deleteDevice(id, config.Vehicles())

			// cleanup references
			for _, dev := range h.Devices() {
				lp := dev.Instance()
				if lp.GetDefaultVehicleRef() == config.NameForID(id) {
					lp.SetDefaultVehicleRef("")
				}
			}

		case templates.Circuit:
			err = deleteDevice(id, config.Circuits())

			// cleanup references
			for _, dev := range h.Devices() {
				lp := dev.Instance()
				if lp.GetCircuitRef() == config.NameForID(id) {
					lp.SetCircuitRef("")
				}
			}
		}

		setConfigDirty()

		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := struct {
			ID int `json:"id"`
		}{
			ID: id,
		}

		jsonResult(w, res)
	}
}

func testConfig[T any](ctx context.Context, id int, class templates.Class, req configReq, newFromConf newFromConfFunc[T], h config.Handler[T]) (T, error) {
	if id == 0 {
		return newFromConf(ctx, req.Type, req.Other)
	}

	_, instance, _, err := deviceInstanceFromMergedConfig(ctx, id, class, req, newFromConf, h)

	return instance, err
}

// testConfigHandler tests a configuration by class
func testConfigHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := templates.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	var id int
	if vars["id"] != "" {
		// test existing device with updated config
		id, err = strconv.Atoi(vars["id"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
	}

	req, err := decodeDeviceConfig(r.Body)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	var instance any
	ctx, cancel, done := startDeviceTimeout()

	switch class {
	case templates.Charger:
		instance, err = testConfig(ctx, id, class, req, charger.NewFromConfig, config.Chargers())

	case templates.Meter:
		instance, err = testConfig(ctx, id, class, req, meter.NewFromConfig, config.Meters())

	case templates.Vehicle:
		instance, err = testConfig(ctx, id, class, req, vehicle.NewFromConfig, config.Vehicles())

	case templates.Circuit:
		err = api.ErrNotAvailable
	}

	if err != nil {
		cancel()
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	// prevent context from being cancelled
	close(done)

	jsonResult(w, testInstance(instance))
}

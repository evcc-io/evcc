package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/meter"
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

// devicesHandler returns a device configurations by class
func devicesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := templates.ClassString(vars["class"])
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
		"type": conf.Type,
	}

	if configurable, ok := dev.(config.ConfigurableDevice[T]); ok {
		// from database
		params, err := sanitizeMasked(class, conf.Other)
		if err != nil {
			return nil, err
		}

		dc["id"] = configurable.ID()
		dc["config"] = params
	} else if title := conf.Other["title"]; title != nil {
		// from yaml- add title only
		if s, ok := title.(string); ok {
			dc["config"] = map[string]any{"title": s}
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
	}

	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

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

// deviceStatusHandler returns a device configuration by class
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
	}

	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	jsonResult(w, testInstance(instance))
}

func newDevice[T any](class templates.Class, req map[string]any, newFromConf func(string, map[string]any) (T, error), h config.Handler[T]) (*config.Config, error) {
	instance, err := newFromConf(typeTemplate, req)
	if err != nil {
		return nil, err
	}

	conf, err := config.AddConfig(class, typeTemplate, req)
	if err != nil {
		return nil, err
	}

	return &conf, h.Add(config.NewConfigurableDevice[T](conf, instance))
}

// newDeviceHandler creates a new device by class
func newDeviceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := templates.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	delete(req, "type")

	var conf *config.Config

	switch class {
	case templates.Charger:
		conf, err = newDevice(class, req, charger.NewFromConfig, config.Chargers())

	case templates.Meter:
		conf, err = newDevice(class, req, meter.NewFromConfig, config.Meters())

	case templates.Vehicle:
		conf, err = newDevice(class, req, vehicle.NewFromConfig, config.Vehicles())
	}

	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

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

func updateDevice[T any](id int, class templates.Class, conf map[string]any, newFromConf func(string, map[string]any) (T, error), h config.Handler[T]) error {
	dev, instance, merged, err := deviceInstanceFromMergedConfig(id, class, conf, newFromConf, h)
	if err != nil {
		return err
	}

	configurable, ok := dev.(config.ConfigurableDevice[T])
	if !ok {
		return errors.New("not configurable")
	}

	return configurable.Update(merged, instance)
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

	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	delete(req, "type")

	switch class {
	case templates.Charger:
		err = updateDevice(id, class, req, charger.NewFromConfig, config.Chargers())

	case templates.Meter:
		err = updateDevice(id, class, req, meter.NewFromConfig, config.Meters())

	case templates.Vehicle:
		err = updateDevice(id, class, req, vehicle.NewFromConfig, config.Vehicles())
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

func deleteDevice[T any](id int, h config.Handler[T]) error {
	name := config.NameForID(id)

	dev, err := h.ByName(name)
	if err != nil {
		return err
	}

	configurable, ok := dev.(config.ConfigurableDevice[T])
	if !ok {
		return errors.New("not configurable")
	}

	if err := configurable.Delete(); err != nil {
		return err
	}

	return h.Delete(name)
}

// deleteDeviceHandler deletes a device from database by class
func deleteDeviceHandler(w http.ResponseWriter, r *http.Request) {
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

	case templates.Meter:
		err = deleteDevice(id, config.Meters())

	case templates.Vehicle:
		err = deleteDevice(id, config.Vehicles())
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

func testConfig[T any](id int, class templates.Class, conf map[string]any, newFromConf func(string, map[string]any) (T, error), h config.Handler[T]) (T, error) {
	if id == 0 {
		return newFromConf(typeTemplate, conf)
	}

	_, instance, _, err := deviceInstanceFromMergedConfig(id, class, conf, newFromConf, h)

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

	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	delete(req, "type")

	var instance any

	switch class {
	case templates.Charger:
		instance, err = testConfig(id, class, req, charger.NewFromConfig, config.Chargers())

	case templates.Meter:
		instance, err = testConfig(id, class, req, meter.NewFromConfig, config.Meters())

	case templates.Vehicle:
		instance, err = testConfig(id, class, req, vehicle.NewFromConfig, config.Vehicles())
	}

	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	jsonResult(w, testInstance(instance))
}

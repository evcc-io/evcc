package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/gorilla/mux"
)

const (
	// typeTemplate is the updatable configuration type
	typeTemplate = "template"

	// masked indicates a masked config parameter value
	masked = "***"
)

// templatesHandler returns the list of templates by class
func templatesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := templates.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	lang := r.URL.Query().Get("lang")
	templates.EncoderLanguage(lang)

	if name := r.URL.Query().Get("name"); name != "" {
		res, err := templates.ByName(class, name)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonResult(w, res)
		return
	}

	jsonResult(w, templates.ByClass(class))
}

// productsHandler returns the list of products by class
func productsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := templates.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	tmpl := templates.ByClass(class)
	lang := r.URL.Query().Get("lang")

	res := make(products, 0)
	for _, t := range tmpl {
		for _, p := range t.Products {
			res = append(res, product{
				Name:     p.Title(lang),
				Template: t.TemplateDefinition.Template,
				Group:    t.Group,
			})
		}
	}

	slices.SortFunc(res, func(a, b product) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

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

// deviceHandler returns a device configuration by class
func deviceHandler(w http.ResponseWriter, r *http.Request) {
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

	res := struct {
		ID int `json:"id"`
	}{
		ID: conf.ID,
	}

	jsonResult(w, res)
}

func updateDevice[T any](id int, class templates.Class, conf map[string]any, newFromConf func(string, map[string]any) (T, error), h config.Handler[T]) error {
	dev, instance, err := deviceInstanceFromMergedConfig(id, class, conf, newFromConf, h)
	if err != nil {
		return err
	}

	configurable, ok := dev.(config.ConfigurableDevice[T])
	if !ok {
		return errors.New("not configurable")
	}

	return configurable.Update(conf, instance)
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

func testDevice[T any](id int, class templates.Class, conf map[string]any, newFromConf func(string, map[string]any) (T, error), h config.Handler[T]) (T, error) {
	if id == 0 {
		return newFromConf(typeTemplate, conf)
	}

	_, instance, err := deviceInstanceFromMergedConfig(id, class, conf, newFromConf, h)

	return instance, err
}

// testHandler tests a configuration by class
func testHandler(w http.ResponseWriter, r *http.Request) {
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
		instance, err = testDevice(id, class, req, charger.NewFromConfig, config.Chargers())

	case templates.Meter:
		instance, err = testDevice(id, class, req, meter.NewFromConfig, config.Meters())

	case templates.Vehicle:
		instance, err = testDevice(id, class, req, vehicle.NewFromConfig, config.Vehicles())
	}

	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	type result = struct {
		Value any   `json:"value"`
		Error error `json:"error"`
	}

	res := make(map[string]result)

	if dev, ok := instance.(api.Meter); ok {
		val, err := dev.CurrentPower()
		res["CurrentPower"] = result{val, err}
	}

	if dev, ok := instance.(api.MeterEnergy); ok {
		val, err := dev.TotalEnergy()
		res["TotalEnergy"] = result{val, err}
	}

	if dev, ok := instance.(api.Battery); ok {
		val, err := dev.Soc()
		res["Soc"] = result{val, err}
	}

	jsonResult(w, res)
}

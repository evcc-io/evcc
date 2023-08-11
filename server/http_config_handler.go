package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/gorilla/mux"
	"golang.org/x/exp/slices"
)

// typeTemplate is the updatable configuration type
const typeTemplate = "template"

// templatesHandler returns the list of templates by class
func templatesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := config.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	res := templates.ByClass(class)

	lang := r.URL.Query().Get("lang")
	templates.EncoderLanguage(lang)

	if name := r.URL.Query().Get("name"); name != "" {
		for _, t := range res {
			if t.TemplateDefinition.Template == name {
				jsonResult(w, t)
				return
			}
		}

		jsonError(w, http.StatusBadRequest, errors.New("template not found"))
		return
	}

	jsonResult(w, res)
}

// productsHandler returns the list of products by class
func productsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := config.ClassString(vars["class"])
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

	slices.SortFunc(res, func(a, b product) bool {
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})

	jsonResult(w, res)
}

func devicesConfig[T any](h config.Handler[T]) []map[string]any {
	var res []map[string]any

	// omit name from config
	for _, dev := range h.Devices() {
		conf := dev.Config()

		dc := map[string]any{
			"name": conf.Name,
			"type": conf.Type,
		}

		if configurable, ok := dev.(config.ConfigurableDevice[T]); ok {
			// from database
			dc["id"] = configurable.ID()
			dc["config"] = conf.Other
		} else if title := conf.Other["title"]; title != nil {
			// from yaml- add title only
			if s, ok := title.(string); ok {
				dc["config"] = map[string]any{"title": s}
			}
		}

		res = append(res, dc)
	}

	return res
}

// devicesHandler tests a configuration by class
func devicesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := config.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	var res []map[string]any

	switch class {
	case config.Meter:
		res = devicesConfig(config.Meters())
	case config.Charger:
		res = devicesConfig(config.Chargers())
	case config.Vehicle:
		res = devicesConfig(config.Vehicles())
	}

	jsonResult(w, res)
}

func newDevice[T any](class config.Class, req map[string]any, newFromConf func(string, map[string]any) (T, error), h config.Handler[T]) (*config.Config, error) {
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

	class, err := config.ClassString(vars["class"])
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
	case config.Charger:
		conf, err = newDevice(class, req, charger.NewFromConfig, config.Chargers())

	case config.Meter:
		conf, err = newDevice(class, req, meter.NewFromConfig, config.Meters())

	case config.Vehicle:
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

func updateDevice[T any](id int, conf map[string]any, newFromConf func(string, map[string]any) (T, error), h config.Handler[T]) error {
	dev, err := h.ByName(config.NameForID(id))
	if err != nil {
		return err
	}

	instance, err := newFromConf(typeTemplate, conf)
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

	class, err := config.ClassString(vars["class"])
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
	case config.Charger:
		err = updateDevice(id, req, charger.NewFromConfig, config.Chargers())

	case config.Meter:
		err = updateDevice(id, req, meter.NewFromConfig, config.Meters())

	case config.Vehicle:
		err = updateDevice(id, req, vehicle.NewFromConfig, config.Vehicles())
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

	class, err := config.ClassString(vars["class"])
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
	case config.Charger:
		err = deleteDevice(id, config.Chargers())

	case config.Meter:
		err = deleteDevice(id, config.Meters())

	case config.Vehicle:
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

// testHandler tests a configuration by class
func testHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := config.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	typ := typeTemplate
	if req["type"] != nil {
		typ = req["type"].(string)
		delete(req, "type")
	}

	var dev any

	switch class {
	case config.Charger:
		dev, err = charger.NewFromConfig(typ, req)
	case config.Meter:
		dev, err = meter.NewFromConfig(typ, req)
	case config.Vehicle:
		dev, err = vehicle.NewFromConfig(typ, req)
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

	if dev, ok := dev.(api.Meter); ok {
		val, err := dev.CurrentPower()
		res["CurrentPower"] = result{val, err}
	}

	if dev, ok := dev.(api.MeterEnergy); ok {
		val, err := dev.TotalEnergy()
		res["TotalEnergy"] = result{val, err}
	}

	if dev, ok := dev.(api.Battery); ok {
		val, err := dev.Soc()
		res["Soc"] = result{val, err}
	}

	jsonResult(w, res)
}

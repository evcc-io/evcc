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

func deviceConfigMap[T any](devices []config.Device[T]) []map[string]any {
	var res []map[string]any

	// omit name from config
	for _, dev := range devices {
		conf := dev.Config()

		dc := map[string]any{
			"name": conf.Name,
			"type": conf.Type,
		}

		if _, ok := dev.(config.ConfigurableDevice[T]); ok {
			// from database
			dc["id"] = conf.ID
			dc["config"] = conf.Other
		} else if title := dev.Other["title"]; title != nil {
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
		res = deviceConfigMap[api.Meter](config.Meters())
	case config.Charger:
		res = deviceConfigMap[api.Charger](config.Chargers())
	case config.Vehicle:
		res = deviceConfigMap[api.Vehicle](config.Vehicles())
	}

	jsonResult(w, res)
}

// namedConfig strips the config of type and name
func namedConfig(req *map[string]any) config.Named {
	res := config.Named{
		Type:  typeTemplate,
		Other: *req,
	}
	if (*req)["type"] != nil {
		res.Type = (*req)["type"].(string)
		delete(*req, "type")
	}
	if (*req)["name"] != nil {
		res.Name = (*req)["name"].(string)
		delete(*req, "name")
	}
	return res
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

	var conf config.Config

	switch class {
	case config.Charger:
		var instance api.Charger
		if instance, err = charger.NewFromConfig(typeTemplate, req); err != nil {
			break
		}

		if conf, err = config.AddConfig(config.Charger, typeTemplate, req); err != nil {
			break
		}

		dev := config.NewConfigurableDevice[api.Charger](conf)
		dev.Connect(instance)
		err = config.AddCharger(dev)

	case config.Meter:
		var instance api.Meter
		if instance, err = meter.NewFromConfig(typeTemplate, req); err != nil {
			break
		}

		if conf, err = config.AddConfig(config.Meter, typeTemplate, req); err != nil {
			break
		}

		dev := config.NewConfigurableDevice[api.Meter](conf)
		dev.Connect(instance)
		err = config.AddMeter(dev)

	case config.Vehicle:
		var instance api.Vehicle
		if instance, err = vehicle.NewFromConfig(typeTemplate, req); err != nil {
			break
		}

		if conf, err = config.AddConfig(config.Vehicle, typeTemplate, req); err != nil {
			break
		}

		dev := config.NewConfigurableDevice[api.Vehicle](conf)
		dev.Connect(instance)
		err = config.AddVehicle(dev)
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

	// named := namedConfig(&req)
	// named.Name = config.NameForID(id)

	switch class {
	case config.Charger:
		var instance api.Charger
		if instance, err = charger.NewFromConfig(typeTemplate, req); err != nil {
			break
		}

		if err = dev.Update(named.Other); err != nil {
			break
		}

		dev.Connect(instance)

	case config.Meter:
		var instance api.Meter
		if instance, err = meter.NewFromConfig(typeTemplate, req); err != nil {
			break
		}

		if err = dev.Update(named.Other); err != nil {
			break
		}

		dev.Connect(instance)

	case config.Vehicle:
		var instance api.Vehicle
		if instance, err = vehicle.NewFromConfig(typeTemplate, req); err != nil {
			break
		}

		if err = dev.Update(named.Other); err != nil {
			break
		}

		dev.Connect(instance)
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

	name := config.NameForID(id)

	switch class {
	case config.Charger:
		if err = config.DeleteCharger(name); err != nil {
			break
		}
		err = config.DeleteDevice(config.Charger, id)

	case config.Meter:
		if err = config.DeleteMeter(name); err != nil {
			break
		}
		err = config.DeleteDevice(config.Meter, id)

	case config.Vehicle:
		if err = config.DeleteVehicle(name); err != nil {
			break
		}
		err = config.DeleteDevice(config.Vehicle, id)
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

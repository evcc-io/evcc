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
			})
		}
	}

	slices.SortFunc(res, func(a, b product) bool {
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})

	jsonResult(w, res)
}

// devicesHandler tests a configuration by class
func devicesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := config.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	var named []config.Named

	switch class {
	case config.Meter:
		named = config.MetersConfig()
	case config.Charger:
		named = config.ChargersConfig()
	case config.Vehicle:
		named = config.VehiclesConfig()
	}

	type deviceConfig map[string]any
	res := make([]deviceConfig, 0, len(named))

	// omit name from config
	for _, v := range named {
		conf := deviceConfig{
			"name": v.Name,
			"type": v.Type,
		}

		// convert name to id
		var id int
		if sid, ok := strings.CutPrefix(v.Name, "db:"); ok {
			if i, err := strconv.Atoi(sid); err == nil {
				id = i
			}
		}

		if id > 0 {
			// from database
			conf["id"] = id
			conf["config"] = v.Other
		} else if title := v.Other["title"]; title != nil {
			// from yaml- add title only
			if s, ok := title.(string); ok {
				conf["config"] = map[string]any{"title": s}
			}
		}

		res = append(res, conf)
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

	var id int
	named := namedConfig(&req)

	switch class {
	case config.Charger:
		var c api.Charger
		if c, err = charger.NewFromConfig(named.Type, req); err != nil {
			break
		}

		if id, err = config.AddDevice(config.Charger, named.Type, req); err != nil {
			break
		}

		named.Name = config.NameForID(id)
		err = config.AddCharger(named, c)

	case config.Meter:
		var m api.Meter
		if m, err = meter.NewFromConfig(named.Type, req); err != nil {
			break
		}

		if id, err = config.AddDevice(config.Meter, named.Type, req); err != nil {
			break
		}

		named.Name = config.NameForID(id)
		err = config.AddMeter(named, m)

	case config.Vehicle:
		var v api.Vehicle
		if v, err = vehicle.NewFromConfig(named.Type, req); err != nil {
			break
		}

		if id, err = config.AddDevice(config.Vehicle, named.Type, req); err != nil {
			break
		}

		named.Name = config.NameForID(id)
		err = config.AddVehicle(named, v)
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

	named := namedConfig(&req)

	switch class {
	case config.Charger:
		var c api.Charger
		if c, err = charger.NewFromConfig(named.Type, req); err != nil {
			break
		}

		if _, err = config.UpdateDevice(config.Charger, id, named.Other); err != nil {
			break
		}

		err = config.UpdateCharger(named, c)

	case config.Meter:
		var m api.Meter
		if m, err = meter.NewFromConfig(named.Type, req); err != nil {
			break
		}

		if _, err = config.UpdateDevice(config.Meter, id, named.Other); err != nil {
			break
		}

		err = config.UpdateMeter(named, m)

	case config.Vehicle:
		var v api.Vehicle
		if v, err = vehicle.NewFromConfig(named.Type, req); err != nil {
			break
		}

		if _, err = config.UpdateDevice(config.Vehicle, id, named.Other); err != nil {
			break
		}

		err = config.UpdateVehicle(named, v)
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
		_, err = config.DeleteDevice(config.Charger, id)

	case config.Meter:
		if err = config.DeleteMeter(name); err != nil {
			break
		}
		_, err = config.DeleteDevice(config.Meter, id)

	case config.Vehicle:
		if err = config.DeleteVehicle(name); err != nil {
			break
		}
		_, err = config.DeleteDevice(config.Vehicle, id)
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

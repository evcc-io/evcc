package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/gorilla/mux"
)

// templatesHandler returns the list of templates by class
func templatesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := templates.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	res := templates.ByClass(class)
	jsonResult(w, res)
}

// testHandler tests a configuration by class
func testHandler(w http.ResponseWriter, r *http.Request) {
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

	// var req struct {
	// 	Type  string
	// 	Other map[string]any `mapstructure:",remain"`
	// }

	// if err := mapstructure.Decode(plain, &req); err != nil {
	// 	jsonError(w, http.StatusBadRequest, err)
	// 	return
	// }

	// tmpl, err := templates.ByName(class, req.Name)
	// if err != nil {
	// 	jsonError(w, http.StatusBadRequest, err)
	// 	return
	// }

	// b, _, err := tmpl.RenderResult(templates.TemplateRenderModeInstance, req.Other)
	// if err != nil {
	// 	jsonError(w, http.StatusBadRequest, err)
	// 	return
	// }

	// var instance any
	// if err := yaml.Unmarshal(b, &instance); err != nil {
	// 	jsonError(w, http.StatusBadRequest, err)
	// 	return
	// }

	typ := "template"
	if req["type"] != nil {
		typ = req["type"].(string)
		delete(req, "type")
	}

	var dev any

	switch class {
	case templates.Charger:
		dev, err = charger.NewFromConfig(typ, req)
	case templates.Meter:
		dev, err = meter.NewFromConfig(typ, req)
	case templates.Vehicle:
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

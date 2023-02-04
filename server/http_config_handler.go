package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
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

	var plain map[string]any
	if err := json.NewDecoder(r.Body).Decode(&plain); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	var req struct {
		Name  string
		Other map[string]any `mapstructure:",remain"`
	}

	if err := mapstructure.Decode(plain, &req); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

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

	var res any

	switch class {
	case templates.Charger:
		// res, err = charger.NewFromConfig(req.Name, req.Other)
	case templates.Meter:
		res, err = meter.NewFromConfig(req.Name, req.Other)
	case templates.Vehicle:
		res, err = vehicle.NewFromConfig(req.Name, req.Other)
	}

	_ = res

	jsonResult(w, req)
}

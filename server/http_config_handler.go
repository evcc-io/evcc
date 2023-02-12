package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/gorilla/mux"
	"golang.org/x/exp/slices"
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

type product struct {
	Name     string `json:"name"`
	Template string `json:"template"`
}

type products []product

func (p products) MarshalJSON() (out []byte, err error) {
	if p == nil {
		return []byte(`null`), nil
	}
	if len(p) == 0 {
		return []byte(`{}`), nil
	}

	out = append(out, '{')
	for _, e := range p {
		key, err := json.Marshal(e.Name)
		if err != nil {
			return nil, err
		}
		val, err := json.Marshal(e.Template)
		if err != nil {
			return nil, err
		}
		out = append(out, key...)
		out = append(out, ':')
		out = append(out, val...)
		out = append(out, ',')
	}

	// replace last ',' with '}'
	if len(out) > 1 {
		out[len(out)-1] = '}'
	} else {
		out = append(out, '}')
	}

	return out, nil
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
			})
		}
	}

	slices.SortFunc(res, func(a, b product) bool {
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})

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

package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/gorilla/mux"
	"github.com/samber/lo"
)

var supportedLanguages = []string{"en", "de"}

func getLang(r *http.Request) string {
	lang := r.URL.Query().Get("lang")
	if !slices.Contains(supportedLanguages, lang) {
		lang = supportedLanguages[0]
	}
	return lang
}

// templatesHandler returns the list of templates by class
func templatesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	class, err := templates.ClassString(vars["class"])
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	lang := getLang(r)
	templates.EncoderLanguage(lang)

	// filter deprecated properties
	filterParams := func(t templates.Template) templates.Template {
		t.Params = lo.Filter(t.Params, func(p templates.Param, _ int) bool {
			return !p.IsDeprecated()
		})
		return t
	}

	if name := r.URL.Query().Get("name"); name != "" {
		res, err := templates.ByName(class, name)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonWrite(w, filterParams(res))
		return
	}

	var res []templates.Template
	for _, t := range templates.ByClass(class) {
		res = append(res, filterParams(t))
	}

	jsonWrite(w, res)
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
	lang := getLang(r)
	usage := r.URL.Query().Get("usage")

	res := make(products, 0)
	for _, t := range tmpl {
		// if usage filter is specified, only include templates with matching usage
		if usage == "" || slices.Contains(t.Usages(), usage) {
			for _, p := range t.Products {
				res = append(res, product{
					Name:     p.Title(lang),
					Template: t.TemplateDefinition.Template,
					Group:    t.Group,
				})
			}
		}
	}

	slices.SortFunc(res, func(a, b product) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	jsonWrite(w, res)
}

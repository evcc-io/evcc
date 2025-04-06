package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/gorilla/mux"
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

	if name := r.URL.Query().Get("name"); name != "" {
		res, err := templates.ByName(class, name)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonResult(w, res)
		return
	}

	// filter deprecated properties
	res := make([]templates.Template, 0)
	for _, t := range templates.ByClass(class) {
		params := make([]templates.Param, 0, len(t.Params))
		for _, p := range t.Params {
			if p.Deprecated == nil || !*p.Deprecated {
				params = append(params, p)
			}
		}
		t.Params = params
		res = append(res, t)
	}

	jsonResult(w, res)
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
		includeUsage := usage == ""
		if !includeUsage {
			for _, u := range t.Usages() {
				if u == usage {
					includeUsage = true
					break
				}
			}
		}

		if includeUsage {
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

	jsonResult(w, res)
}

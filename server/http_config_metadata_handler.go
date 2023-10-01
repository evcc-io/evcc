package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util/templates"
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

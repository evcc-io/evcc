package server

import (
	"net/http"

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

	res := templates.ByClass(class)
	jsonResult(w, res)
}

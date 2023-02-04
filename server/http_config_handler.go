package server

import (
	"net/http"

	"github.com/evcc-io/evcc/util/templates"
)

// configHandler returns the list of charging sessions
func configHandler(w http.ResponseWriter, r *http.Request) {
	res := templates.ByClass(templates.Meter)
	jsonResult(w, res)
}
